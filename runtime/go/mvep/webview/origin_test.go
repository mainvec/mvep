package webview

import (
	"encoding/base64"
	"testing"
)

func TestWailsOrigin(t *testing.T) {
	for origin, want := range map[string]bool{
		"wails://wails":           true,
		"wails://localhost":       true,
		"wails://localhost:3145":  true,
		"http://wails.localhost":  true,
		"https://wails.localhost": true,
		"https://WAILS.localhost": true,
		// Dev mode serves the page on the dev server's port (#68).
		"http://wails.localhost:3145":  true,
		"https://wails.localhost:3145": true,
		"http://wails.localhost:80":    true,
		// Look-alikes stay refused, with or without a port.
		"https://wails.localhost.io":        false,
		"https://wails.localhost.io:3145":   false,
		"https://evil.wails.localhost":      false,
		"https://evil.wails.localhost:3145": false,
		"http://wails.localhost:3145/path":  false,
		"http://wails.localhost:abc":        false,
		"https://attacker.example":          false,
		"https://attacker.example:3145":     false,
		"wails://wails/path":                false,
		"null":                              false,
		"":                                  false,
		"wails:":                            false,
	} {
		if got := WailsOrigin(origin); got != want {
			t.Errorf("WailsOrigin(%q) = %v, want %v", origin, got, want)
		}
	}
}

func TestCapacitorOrigin(t *testing.T) {
	for origin, want := range map[string]bool{
		"capacitor://localhost":    true,
		"https://localhost":        true,
		"http://localhost":         false,
		"https://localhost:8100":   false,
		"capacitor://localhost.x":  false,
		"https://attacker.example": false,
		"null":                     false,
	} {
		if got := CapacitorOrigin(origin); got != want {
			t.Errorf("CapacitorOrigin(%q) = %v, want %v", origin, got, want)
		}
	}
}

func TestNewToken(t *testing.T) {
	a, b := NewToken(), NewToken()
	if a == b {
		t.Fatal("two tokens are equal")
	}
	raw, err := base64.RawURLEncoding.DecodeString(a)
	if err != nil || len(raw) != 32 {
		t.Fatalf("token %q decodes to %d bytes (err %v), want 32", a, len(raw), err)
	}
}
