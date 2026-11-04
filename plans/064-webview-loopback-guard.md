# Plan 064 — `webview`: a guarded loopback proxy for daemon + webview apps

**GitHub Issue**: [#64](https://github.com/mainvec/mvep/issues/64)

## Progress

- [x] T1: Export the mvep wire header names; use them in transport + server
- [x] T2: `webview.Guard` (Host → Origin → token, CORS, preflight)
- [x] T3: `WailsOrigin` / `CapacitorOrigin` / `NewToken`
- [x] T4: `webview.Start` streaming reverse proxy (Unix socket or TCP daemon)
- [x] T5: `webviewtest` conformance check
- [x] T6: README (proxy + in-process) and CHANGELOG

## Problem / Goal

Apps that pair a local mvep daemon with a webview (Wails, Capacitor) each
hand-write the loopback listener the page talks to. Getting it safe is subtle
(DNS rebinding, browser pages reaching `127.0.0.1`, other apps on a phone
reaching the port, CORS not preventing a request from being *sent*), and the
CORS header lists drift from mvep's own headers. Write it once in `runtime/go`.

## Fact-check against the code (deviations from the issue text)

- **Error headers are `x-mainvec-error` / `x-mainvec-error-code`**, not
  `x-mvep-error*` (`http_transport.go` `WriteCmdResp`). The command header is
  `x-mainvec-cmd`. Auth is `x-mvep-auth` (`CmdReq.Headers["auth"]` with
  `HeaderPrefix`). Response metadata is any `x-mvep-*` header, set dynamically
  (`job-id`, `job-status`, `job-progress-*`, `job-encoder`, interceptor headers).
- **The server's own CORS never sets `Access-Control-Expose-Headers`**, so a
  page can't read `x-mvep-*` or the error code today. Out of scope to change
  the server's CORS here, but the guard must expose them.
- These names exist only as string literals in `http_transport.go` and
  `server/server.go`, which is the drift the issue describes.

## Design

Package `runtime/go/mvep/webview`, plus `runtime/go/mvep/webview/webviewtest`
(the `net/http/httptest` naming convention).

### T1 — Header constants

Add to `mvep/envelope.go`: `HeaderCmd = "x-mainvec-cmd"`,
`HeaderError = "x-mainvec-error"`, `HeaderErrorCode = "x-mainvec-error-code"`,
`HeaderAuth = HeaderPrefix + "auth"`. Replace the literals in
`http_transport.go` and `server/server.go` (mechanical, no behaviour change).

### T2 — `Guard(next http.Handler, opts GuardOptions) http.Handler`

```go
type GuardOptions struct {
    Token       string                  // required; Guard panics if empty
    AllowOrigin func(origin string) bool // nil admits no Origin at all
}
```

Checks, in order. A refusal never carries CORS headers for a foreign origin.

1. **Host.** `r.Host` must equal the address the connection was accepted on
   (`http.LocalAddrContextKey`), and that address must be a loopback IP.
   Deriving it from the connection means the caller never configures the port,
   and a request with no local address (or a Unix-socket/non-loopback listener)
   fails closed. A rebound domain sends `Host: evil.example:<port>` → 403.
2. **Origin.** If present it must pass `AllowOrigin`, else 403. `"null"` is
   never admitted by the provided checkers.
3. **Preflight** (`OPTIONS` + `Origin` + `Access-Control-Request-Method`) is
   answered by the guard after steps 1–2 and never forwarded. It carries no
   credentials. Responds 204 with the exact origin, `Vary: Origin`,
   `Allow-Methods: GET, POST`, `Allow-Headers` = the requested headers filtered
   to `Content-Type`, `Authorization`, `x-mainvec-cmd`, `x-mvep-*`, a `Max-Age`,
   and `Access-Control-Allow-Private-Network: true` when Chrome's Private
   Network Access asks for it.
4. **Token.** `Authorization: Bearer <t>` or `x-mvep-auth: <t>`, compared in
   constant time (SHA-256 of both sides, then `subtle.ConstantTimeCompare`, so
   length doesn't leak either). Missing/wrong → 401. An admitted origin still
   gets CORS on the 401 so the page can read why.

On admitted responses with an Origin: `Access-Control-Allow-Origin: <exact>`,
never `*`. `Vary: Origin` is always added. `Access-Control-Expose-Headers` is
**computed per response** from the `x-mvep-*` / `x-mainvec-*` headers actually
being written (a `ResponseWriter` wrapper that implements `Flush` and `Unwrap`,
so streaming and `http.ResponseController` keep working). No hard-coded list,
so it cannot drift. Refusals set `x-mainvec-error-code` (`forbidden` /
`unauthorized`), matching `HTTPStatusForErrorCode`.

**In-process use**: the guard does not mark the request locally trusted and
doesn't touch `AuthInterceptor`. If the request used `Bearer`, the guard also
sets `x-mvep-auth` to the same token, so an app's `AuthInterceptor` sees one
credential either way. Its validator accepts the session token.

### T3 — Origin checkers and token

- `WailsOrigin`: scheme `wails` (any host; no web page can hold that scheme),
  or `http`/`https` with host exactly `wails.localhost` and no port (Windows
  WebView2).
- `CapacitorOrigin`: exactly `capacitor://localhost` (iOS) or
  `https://localhost` (Android, Capacitor ≥ 6 default). Apps with a custom
  scheme pass their own func.
- `NewToken() string`: 32 random bytes, base64url, no padding.

### T4 — `Start(opts ProxyOptions) (*Proxy, error)`

```go
type ProxyOptions struct {
    Daemon      string                  // "unix:///path.sock" | "http://127.0.0.1:7000" | "127.0.0.1:7000"
    DaemonToken string                  // sent as x-mvep-auth; empty for a trusted Unix socket
    AllowOrigin func(origin string) bool // required
    Addr        string                  // listen address, default "127.0.0.1:0"; must be a loopback IP
}
type Proxy struct{ URL, Token string /* + unexported */ }
func (p *Proxy) Close() error
```

`Guard` in front of `httputil.ReverseProxy` (`Rewrite`, `FlushInterval: -1`
for streaming; request context cancellation propagates to the daemon). The
rewrite drops `Origin`, `Referer`, `Cookie`, `Authorization` and the
session `x-mvep-auth`, then sets `x-mvep-auth: DaemonToken` if given. So the
page never holds the daemon token and the daemon sees no browser origin.
`ModifyResponse` strips upstream `Access-Control-*`, so only the guard's CORS
applies. A daemon dial failure → 502 with `x-mainvec-error-code:
daemon_unavailable`. A `http.Server` with `ReadHeaderTimeout` serves it.
`Close` closes listener, connections and idle upstream conns.

### T5 — `webviewtest`

```go
type Config struct {
    URL, Token, Origin string // Origin = the app's own origin
    Path   string             // default "/"
    Client *http.Client       // default: 10s timeout
}
func Check(t *testing.T, cfg Config) // one subtest per rule
func Verify(ctx context.Context, cfg Config) error // same rules, no testing.T
```

Rules: no token → refused; wrong token → refused; foreign `Origin` with a valid
token → refused and no ACAO; foreign `Host` with a valid token → refused;
foreign preflight → refused and no ACAO; own origin with token → admitted,
ACAO is the exact origin, `Vary` includes `Origin`; own preflight → admitted
with exact ACAO; ACAO is never `*`. "Refused" means status 401, 403 or 421;
"admitted" means anything else (the daemon may legitimately 400/404 a probe).

### T6 — Docs

`mvep/webview/README.md` with proxy and in-process setups, plus a Wails
binding and Capacitor plugin sketch for handing `URL`/`Token` to the page.
Feature bullet in `runtime/go/README.md`, and a `CHANGELOG.md` entry.

## Tasks

### T1–T6

**Verification**: `go build ./... && go vet ./... && go test ./...` in
`runtime/go`. Each guard rule has a table test. Proxy tests cover a Unix-socket
daemon, a TCP daemon, token substitution and `Origin` drop as seen by the
daemon, streaming (first chunk read before the handler returns), client
cancellation reaching the daemon's context, upstream CORS stripped, and 502 on
a dead daemon. `webviewtest.Check` passes against both `Start` and an
in-process `Guard` + `server.Server`, and `Verify` fails against an unguarded
listener. An end-to-end test drives a real `server.Server` through the proxy
with `client.Client`.

## Out of scope

- TLS on the loopback listener, and TLS to a remote daemon (per the issue).
- Changing `server.Server`'s own CORS (e.g. adding Expose-Headers there).
  Worth a follow-up issue.
- Releasing: tag `runtime/go/v0.13.0` and the toolkit bump after merge.

## Risks and Compatibility

- Purely additive apart from T1, which only replaces literals with constants
  of the same value.
- `Guard` panics on an empty token at construction. That's a programmer error,
  and failing at startup beats a guard that admits everyone.
