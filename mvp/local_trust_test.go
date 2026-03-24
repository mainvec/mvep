package mvp

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

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !capturedTrusted {
		t.Error("handler behind LocalTrustMiddleware should see a locally trusted context")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
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
