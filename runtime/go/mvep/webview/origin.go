package webview

import (
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"strings"
)

// WailsOrigin reports whether origin is a Wails webview: the wails:// scheme
// (macOS, Linux), or http(s)://wails.localhost (Windows WebView2, Android).
// No web page can hold the wails scheme, so any host under it is accepted.
//
// wails.localhost is accepted with any port, because in dev mode Wails serves
// the page on the dev server's port (http://wails.localhost:3145). A .localhost
// name only reaches this machine, so the port adds no trust either way; the
// session token is what Guard relies on.
func WailsOrigin(origin string) bool {
	u, ok := parseOrigin(origin)
	if !ok {
		return false
	}
	switch u.Scheme {
	case "wails":
		return true
	case "http", "https":
		return strings.EqualFold(u.Hostname(), "wails.localhost")
	}
	return false
}

// CapacitorOrigin reports whether origin is a Capacitor webview with its
// default scheme: capacitor://localhost (iOS) or https://localhost (Android,
// Capacitor 6 and later). An app with a custom scheme or hostname passes its
// own check instead.
func CapacitorOrigin(origin string) bool {
	return origin == "capacitor://localhost" || origin == "https://localhost"
}

// parseOrigin parses an Origin header value: scheme and host only, with no
// path, query, fragment or user info. The opaque origin "null" is rejected.
func parseOrigin(origin string) (*url.URL, bool) {
	if origin == "" || origin == "null" {
		return nil, false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil ||
		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return nil, false
	}
	return u, true
}

// NewToken returns a random session token: 32 bytes from crypto/rand,
// base64url-encoded without padding.
func NewToken() string {
	b := make([]byte, 32)
	// crypto/rand.Read never returns an error; it crashes the program instead.
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
