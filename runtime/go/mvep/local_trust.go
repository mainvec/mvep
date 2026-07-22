package mvep

import (
	"context"
	"net/http"
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

// LocalTrustMiddleware is an HTTP middleware that marks every incoming request
// as locally trusted. Attach it to listeners that are inherently trusted
// (e.g. Unix domain sockets where filesystem permissions provide authorization).
func LocalTrustMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(ContextWithLocalTrust(r.Context())))
	})
}
