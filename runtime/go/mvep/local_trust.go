package mvep

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
)

type localTrustContextKey struct{}

// ContextWithLocalTrust returns a copy of ctx with the local-trust marker set.
func ContextWithLocalTrust(ctx context.Context) context.Context {
	return context.WithValue(ctx, localTrustContextKey{}, true)
}

// IsLocalTrusted reports whether ctx carries the local-trust marker.
func IsLocalTrusted(ctx context.Context) bool {
	v, _ := ctx.Value(localTrustContextKey{}).(bool)
	return v
}

// LocalTrustMiddleware marks a request as locally trusted only when the peer is
// verifiably local: a Unix-socket peer (empty or abstract RemoteAddr) or a TCP
// peer whose address is loopback. Anything else is not marked trusted and the
// rejection is logged, so a misconfigured listener fails closed rather than
// silently bypassing AuthInterceptor.
//
// AuthInterceptor returns early for trusted contexts, so this middleware is the
// entire authorization boundary on a trusted listener. Attach it only to
// listeners that are intended to be local-only.
func LocalTrustMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLocalPeer(r.RemoteAddr) {
			slog.Warn("local trust rejected non-local peer", "remote_addr", r.RemoteAddr)
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(ContextWithLocalTrust(r.Context())))
	})
}

// isLocalPeer reports whether remoteAddr belongs to a local peer.
//
//   - Unix sockets: RemoteAddr is empty (or "@" for abstract sockets) — trusted.
//   - TCP: trusted only when the host is loopback (127.0.0.0/8, ::1).
//   - Anything else (unparseable, non-loopback): not trusted.
func isLocalPeer(remoteAddr string) bool {
	// Unix domain socket peer.
	if remoteAddr == "" || remoteAddr == "@" || strings.HasPrefix(remoteAddr, "@") {
		return true
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// Might be a bare IP without a port.
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}
