// Package webview guards a loopback HTTP listener so that only an app's own
// webview can reach the mvep daemon behind it.
//
// Loopback is not private: a page in the user's browser can send requests to
// 127.0.0.1, a domain rebound to 127.0.0.1 makes them same-origin, and on a
// phone other apps can connect to an app's loopback port. CORS does not stop a
// request from being sent, only its response from being read. Guard therefore
// checks the Host, then the Origin, then a per-launch session token, before
// anything reaches the daemon.
//
// Use Start to run Guard in front of a reverse proxy to a separate daemon, or
// Guard directly as listener middleware when the daemon runs in-process.
package webview

import (
	"crypto/sha256"
	"crypto/subtle"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// GuardOptions configures Guard.
type GuardOptions struct {
	// Token is the session token a request must carry, as
	// "Authorization: Bearer <token>" or "x-mvep-auth: <token>". Required;
	// see NewToken.
	Token string
	// AllowOrigin reports whether a request's Origin header is the app's own
	// webview, for example WailsOrigin or CapacitorOrigin. Requests without an
	// Origin header skip this check but still need the token. When nil, every
	// request that carries an Origin is refused.
	AllowOrigin func(origin string) bool
}

// preflightMaxAge is how long, in seconds, a browser may cache a preflight.
const preflightMaxAge = 600

// Guard admits a request to next only when all of these hold, in order:
//
//   - Host is the loopback address the connection was accepted on, which
//     defeats DNS rebinding. A request on a non-loopback or Unix-socket
//     listener is refused.
//   - Origin, if present, passes opts.AllowOrigin. CORS echoes only that exact
//     origin, with Vary: Origin, and never answers "*".
//   - It carries opts.Token, compared in constant time.
//
// A CORS preflight carries no credentials, so it is checked on Host and Origin
// only and answered by Guard without reaching next.
//
// Refusals are 403 (Host, Origin) or 401 (token), with x-mainvec-error-code
// set to "forbidden" or "unauthorized". Guard sets Access-Control-Expose-Headers
// on each admitted response to the x-mvep-* and x-mainvec-* headers it carries,
// so the page can read command metadata and error codes.
//
// When the token arrived as a Bearer credential, Guard also sets x-mvep-auth to
// it, so an AuthInterceptor behind an in-process Guard sees one credential
// either way.
//
// Guard panics if opts.Token is empty.
func Guard(next http.Handler, opts GuardOptions) http.Handler {
	if opts.Token == "" {
		panic("webview: Guard requires a non-empty Token")
	}
	if next == nil {
		panic("webview: Guard requires a non-nil handler")
	}
	return &guard{next: next, allowOrigin: opts.AllowOrigin, tokenSum: sha256.Sum256([]byte(opts.Token))}
}

type guard struct {
	next        http.Handler
	allowOrigin func(string) bool
	tokenSum    [sha256.Size]byte
}

func (g *guard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Vary", "Origin")

	if !hostMatchesListener(r) {
		slog.Warn("webview: refused request with foreign Host", "host", r.Host, "remote_addr", r.RemoteAddr)
		refuse(w, http.StatusForbidden, "forbidden", "host not allowed")
		return
	}

	origin, hasOrigin := originOf(r)
	if hasOrigin && (g.allowOrigin == nil || !g.allowOrigin(origin)) {
		slog.Warn("webview: refused request with foreign Origin", "origin", origin)
		refuse(w, http.StatusForbidden, "forbidden", "origin not allowed")
		return
	}

	if hasOrigin && isPreflight(r) {
		g.preflight(w, r, origin)
		return
	}

	if hasOrigin {
		// Echo the admitted origin before the token check, so the app's own
		// page can read a 401 and tell a stale token from a dead daemon.
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w = &exposeWriter{ResponseWriter: w}
	}

	token, bearer := tokenOf(r)
	if !g.tokenMatches(token) {
		refuse(w, http.StatusUnauthorized, "unauthorized", "missing or invalid session token")
		return
	}
	if bearer {
		r.Header.Set(mvep.HeaderAuth, token)
	}

	g.next.ServeHTTP(w, r)
}

// preflight answers a CORS preflight from an admitted origin.
func (g *guard) preflight(w http.ResponseWriter, r *http.Request, origin string) {
	h := w.Header()
	h.Set("Access-Control-Allow-Origin", origin)
	h.Set("Access-Control-Allow-Methods", "GET, POST")
	if allowed := allowedRequestHeaders(r.Header.Get("Access-Control-Request-Headers")); allowed != "" {
		h.Set("Access-Control-Allow-Headers", allowed)
	}
	h.Set("Access-Control-Max-Age", strconv.Itoa(preflightMaxAge))
	// Chrome's Private Network Access asks before a page reaches loopback.
	if strings.EqualFold(r.Header.Get("Access-Control-Request-Private-Network"), "true") {
		h.Set("Access-Control-Allow-Private-Network", "true")
	}
	w.WriteHeader(http.StatusNoContent)
}

// tokenMatches compares token to the session token in constant time. Both
// sides are hashed first so the comparison does not leak the token's length.
func (g *guard) tokenMatches(token string) bool {
	if token == "" {
		return false
	}
	sum := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(sum[:], g.tokenSum[:]) == 1
}

// hostMatchesListener reports whether r.Host names the loopback address the
// connection was accepted on.
func hostMatchesListener(r *http.Request) bool {
	addr, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok || addr == nil {
		return false
	}
	tcp, ok := addr.(*net.TCPAddr)
	if !ok || !tcp.IP.IsLoopback() {
		return false
	}
	return strings.EqualFold(r.Host, tcp.String())
}

// originOf returns the request's Origin header and whether one was sent.
func originOf(r *http.Request) (string, bool) {
	v, ok := r.Header["Origin"]
	if !ok || len(v) == 0 {
		return "", false
	}
	return v[0], true
}

func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
}

// tokenOf returns the session token a request carries and whether it came as
// a Bearer credential. Authorization takes precedence over x-mvep-auth.
func tokenOf(r *http.Request) (token string, bearer bool) {
	if auth := r.Header.Get("Authorization"); auth != "" {
		scheme, rest, ok := strings.Cut(auth, " ")
		if ok && strings.EqualFold(scheme, "Bearer") {
			return strings.TrimSpace(rest), true
		}
		return "", false
	}
	return r.Header.Get(mvep.HeaderAuth), false
}

// allowedRequestHeaders filters a preflight's Access-Control-Request-Headers
// to the headers an mvep client sends: Content-Type, Authorization, the
// command header and any x-mvep-* header.
func allowedRequestHeaders(requested string) string {
	var out []string
	for _, name := range strings.Split(requested, ",") {
		name = strings.ToLower(strings.TrimSpace(name))
		switch {
		case name == "":
		case name == "content-type", name == "authorization", name == mvep.HeaderCmd,
			strings.HasPrefix(name, mvep.HeaderPrefix):
			out = append(out, name)
		}
	}
	return strings.Join(out, ", ")
}

// refuse writes a refusal. It never adds CORS headers, so a foreign page
// cannot read it.
func refuse(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set(mvep.HeaderErrorCode, code)
	http.Error(w, msg, status)
}

// exposeWriter sets Access-Control-Expose-Headers to the mvep headers present
// when the response header is written, so the list cannot drift from what the
// daemon actually sends.
type exposeWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (e *exposeWriter) WriteHeader(code int) {
	if !e.wroteHeader {
		e.wroteHeader = true
		h := e.Header()
		var names []string
		for k := range h {
			lk := strings.ToLower(k)
			if strings.HasPrefix(lk, mvep.HeaderPrefix) || strings.HasPrefix(lk, "x-mainvec-") {
				names = append(names, lk)
			}
		}
		if len(names) > 0 {
			slices.Sort(names)
			h.Set("Access-Control-Expose-Headers", strings.Join(names, ", "))
		}
	}
	e.ResponseWriter.WriteHeader(code)
}

func (e *exposeWriter) Write(b []byte) (int, error) {
	if !e.wroteHeader {
		e.WriteHeader(http.StatusOK)
	}
	return e.ResponseWriter.Write(b)
}

// Flush supports streaming responses through the guard.
func (e *exposeWriter) Flush() {
	if !e.wroteHeader {
		e.WriteHeader(http.StatusOK)
	}
	_ = http.NewResponseController(e.ResponseWriter).Flush()
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (e *exposeWriter) Unwrap() http.ResponseWriter { return e.ResponseWriter }
