# webview

A guarded loopback listener for apps that pair an mvep daemon with a webview
front end, such as a Wails desktop app or a Capacitor mobile app.

```go
import "github.com/mainvec/mvep/runtime/go/mvep/webview"
```

## Why a guard

The page in the webview reaches the daemon over `http://127.0.0.1:<port>`.
That port isn't private:

- **Any web page in the user's browser can send requests to `127.0.0.1`.** CORS
  stops it from *reading* the response, not from *sending* the request.
- **DNS rebinding** points a domain at `127.0.0.1`, which makes the attacker's
  requests same-origin.
- **On a phone, other apps can connect** to your app's loopback port.

`webview.Guard` admits a request only when all three checks pass, in this
order:

1. **Host** is the loopback address the connection was accepted on. A rebound
   domain sends `Host: attacker.example:<port>` and is refused.
2. **Origin**, if present, is your app's webview. CORS echoes only that exact
   origin, with `Vary: Origin`, and never answers `*`.
3. **A random per-launch session token** is present, sent as
   `Authorization: Bearer <token>` or `x-mvep-auth: <token>` and compared in
   constant time.

A CORS preflight carries no credentials, so it is checked on Host and Origin
only, and the guard answers it itself. Admitted responses expose the
`x-mvep-*` and `x-mainvec-*` headers they carry (job status, error codes)
through `Access-Control-Expose-Headers`, computed per response so the list
cannot drift.

| Refusal | Status | `x-mainvec-error-code` |
| --- | --- | --- |
| Foreign Host, or a non-loopback / Unix-socket listener | 403 | `forbidden` |
| Foreign Origin | 403 | `forbidden` |
| Missing or wrong token | 401 | `unauthorized` (readable by your own origin) |

## Setup 1: proxy to a separate daemon

Use this when the daemon is its own process, for example reached over a Unix
socket. `webview.Start` runs the guard in front of a streaming reverse proxy:

```go
p, err := webview.Start(webview.ProxyOptions{
    Daemon:      "unix:///run/user/1000/myapp.sock", // or "http://127.0.0.1:7000"
    DaemonToken: "",                                 // empty: the daemon trusts its socket
    AllowOrigin: webview.WailsOrigin,
})
if err != nil {
    return err
}
defer p.Close()
// Hand p.URL and p.Token to the page (see below).
```

The proxy replaces the session token with `DaemonToken` (sent as
`x-mvep-auth`). It drops `Origin`, `Referer`, `Cookie` and `Authorization`, so
the page never holds the daemon's token and the daemon never sees a browser
origin. Daemon CORS headers are stripped. Responses stream as they arrive, and
a request the page cancels is cancelled at the daemon. A daemon that can't be
reached answers 502 with `x-mainvec-error-code: daemon_unavailable`.

`Addr` defaults to `127.0.0.1:0` (a free port) and must be a loopback IP.

## Setup 2: daemon in the same process

Use `Guard` as listener middleware. Its session token is also the daemon
credential:

```go
token := webview.NewToken()
ln, err := net.Listen("tcp", "127.0.0.1:0")
if err != nil {
    return err
}

svr, err := server.NewServer(&server.ServerConfig{
    Listeners: []server.ListenerConfig{{
        Listener: ln,
        Middleware: func(h http.Handler) http.Handler {
            return webview.Guard(h, webview.GuardOptions{Token: token, AllowOrigin: webview.WailsOrigin})
        },
    }},
    // Optional: a validator that accepts the session token. Guard copies a
    // Bearer token into x-mvep-auth, so AuthInterceptor sees it either way.
    Interceptor: mvep.AuthInterceptor(sessionValidator(token)),
})
// Register packages, StartAsync, then hand "http://"+ln.Addr().String() and
// token to the page.
```

Don't set `EnableCORS` on that server: the guard owns CORS for this listener.

## Handing URL and token to the page

Pass them through your app's own bridge, never through a URL or file the page
loads.

**Wails** (bound method):

```go
type Daemon struct{ URL, Token string }

func (a *App) Daemon() Daemon { return Daemon{URL: a.proxy.URL, Token: a.proxy.Token} }
```

```js
const { URL, Token } = await window.go.main.App.Daemon();
await fetch(`${URL}/api/mypkg/cmd`, {
  method: "POST",
  headers: { "Authorization": `Bearer ${Token}`, "Content-Type": "application/json", "x-mainvec-cmd": "EchoCmd" },
  body: JSON.stringify({ msg: "hi" }),
});
```

**Capacitor**: return `{ url, token }` from a small native plugin method that
calls into the Go side (for example through gomobile). The page uses them the
same way.

## Origin checks

| Func | Admits |
| --- | --- |
| `WailsOrigin` | `wails://…` (macOS, Linux); `http(s)://wails.localhost` (Windows) |
| `CapacitorOrigin` | `capacitor://localhost` (iOS); `https://localhost` (Android, Capacitor 6+) |

Any `func(origin string) bool` works, for a custom scheme or hostname. The
opaque origin `null` is never admitted by the provided checks.

## Conformance check in CI

`webviewtest` checks your real listener against every rule above:

```go
func TestWebviewListener(t *testing.T) {
    p := startMyAppProxy(t)
    webviewtest.Check(t, webviewtest.Config{
        URL:    p.URL,
        Token:  p.Token,
        Origin: "wails://wails", // your app's own origin
        Path:   "/api/mypkg/cmd",
    })
}
```

It asserts that requests without the token, with a wrong token, from a foreign
`Origin`, under a foreign `Host`, and foreign preflights are refused (401, 403
or 421, not readable by the requesting page). It also asserts that your own
origin is admitted with that exact origin echoed and `Vary: Origin`.
`webviewtest.Verify` runs the same rules without `testing.T` and returns the
failures.

## Out of scope

- **TLS on the loopback listener.** Reading loopback traffic takes rights that
  already expose the app's memory, and a self-signed certificate would need the
  OS trust store or per-webview overrides.
- **TLS to a remote daemon.** `Daemon` accepts `https://`, but certificate
  configuration is not handled here.
