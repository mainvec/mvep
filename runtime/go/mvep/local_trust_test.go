package mvep

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContextWithLocalTrust(t *testing.T) {
	ctx := context.Background()
	if IsLocalTrusted(ctx) {
		t.Error("plain context should not be locally trusted")
	}

	ctx = ContextWithLocalTrust(ctx)
	if !IsLocalTrusted(ctx) {
		t.Error("context with local trust should be locally trusted")
	}
}

func TestLocalTrustMiddleware(t *testing.T) {
	var capturedTrusted bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTrusted = IsLocalTrusted(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := LocalTrustMiddleware(inner)

	// httptest.NewRequest sets RemoteAddr to "192.0.2.1:1234" (non-loopback).
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !capturedTrusted {
		t.Error("loopback peer should be locally trusted")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

// TestLocalTrustMiddleware_PeerVerification covers the T7 peer checks.
func TestLocalTrustMiddleware_PeerVerification(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		wantTrust  bool
	}{
		{"unix socket empty addr", "", true},
		{"unix abstract socket", "@/tmp/mvep.sock", true},
		{"loopback ipv4", "127.0.0.1:8080", true},
		{"loopback ipv6", "[::1]:8080", true},
		{"non-loopback ipv4", "192.0.2.10:443", false},
		{"non-loopback lan", "10.1.2.3:443", false},
		{"unparseable", "not-an-address", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var trusted bool
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				trusted = IsLocalTrusted(r.Context())
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tc.remoteAddr
			LocalTrustMiddleware(inner).ServeHTTP(httptest.NewRecorder(), req)
			if trusted != tc.wantTrust {
				t.Errorf("RemoteAddr %q: trusted = %v, want %v", tc.remoteAddr, trusted, tc.wantTrust)
			}
		})
	}
}

func TestLocalTrustMiddleware_DoesNotAffectUntrustedRequests(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Serve directly without LocalTrustMiddleware
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	inner.ServeHTTP(rr, req)

	if IsLocalTrusted(req.Context()) {
		t.Error("plain request should not be locally trusted")
	}
}
