package mvep

import (
	"context"
	"log/slog"
	"time"
)

// LoggingInterceptor logs command execution with timing information
func LoggingInterceptor() CmdInterceptor {
	return func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		start := time.Now()
		slog.Info("cmd.start", "cmd", req.Cmd)

		resp := next(ctx, req)

		duration := time.Since(start)
		if resp.HasError() {
			slog.Error("cmd.end",
				"cmd", req.Cmd,
				"duration", duration,
				"error_code", resp.Error.Code,
				"error_msg", resp.Error.Message,
			)
		} else {
			slog.Info("cmd.end",
				"cmd", req.Cmd,
				"duration", duration,
			)
		}
		return resp
	}
}

// TokenValidator is the interface for validating authentication tokens
type TokenValidator interface {
	// Validate checks if the token is valid and returns user info
	// Returns an error if the token is invalid
	Validate(ctx context.Context, token string) (user any, err error)
}

// userContextKey is the context key for storing user info
type userContextKey struct{}

// AuthInterceptor validates the "auth" header using the provided validator
// On success, it stores the user info in the context accessible via UserFromContext.
// Requests marked as locally trusted (see LocalTrustMiddleware) bypass token validation.
func AuthInterceptor(validator TokenValidator) CmdInterceptor {
	return func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		// Skip auth for locally trusted connections (e.g. Unix socket).
		if IsLocalTrusted(ctx) {
			return next(ctx, req)
		}

		token := req.Headers["auth"]
		if token == "" {
			return NewCmdRespError("unauthorized", "missing auth token")
		}

		user, err := validator.Validate(ctx, token)
		if err != nil {
			return NewCmdRespError("unauthorized", err.Error())
		}

		// Store user in context for handlers to access
		ctx = context.WithValue(ctx, userContextKey{}, user)
		return next(ctx, req)
	}
}

// UserFromContext retrieves the user info set by AuthInterceptor
// Returns nil if no user info is present
func UserFromContext(ctx context.Context) any {
	return ctx.Value(userContextKey{})
}

// RecoveryInterceptor recovers from panics in command handlers
// and returns an error response instead of crashing the server
func RecoveryInterceptor() CmdInterceptor {
	return func(ctx context.Context, req *CmdReq, next CmdHandler) (resp *CmdResp) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("cmd.panic",
					"cmd", req.Cmd,
					"panic", r,
				)
				resp = NewCmdRespError("internal_error", "internal server error")
			}
		}()
		return next(ctx, req)
	}
}

// RequestIDInterceptor ensures every request has a request ID
// If not present in headers, generates one and adds it to both request and response
func RequestIDInterceptor(generator func() string) CmdInterceptor {
	return func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		requestID := req.Headers["request-id"]
		if requestID == "" && generator != nil {
			requestID = generator()
			if req.Headers == nil {
				req.Headers = make(map[string]string)
			}
			req.Headers["request-id"] = requestID
		}

		resp := next(ctx, req)

		// Echo request ID in response
		if requestID != "" {
			resp.WithHeader("request-id", requestID)
		}
		return resp
	}
}

// =============================================================================
// Client-side Interceptors
// =============================================================================

// ClientLoggingInterceptor logs client requests and responses with timing
func ClientLoggingInterceptor() ClientInterceptor {
	return func(ctx context.Context, req *CmdReq, invoker ClientInvoker) (*CmdResp, error) {
		start := time.Now()
		slog.Info("client.request", "cmd", req.Cmd)

		resp, err := invoker(ctx, req)

		duration := time.Since(start)
		if err != nil {
			slog.Error("client.response",
				"cmd", req.Cmd,
				"duration", duration,
				"error", err.Error(),
			)
		} else if resp != nil && resp.HasError() {
			slog.Error("client.response",
				"cmd", req.Cmd,
				"duration", duration,
				"error_code", resp.Error.Code,
				"error_msg", resp.Error.Message,
			)
		} else {
			slog.Info("client.response",
				"cmd", req.Cmd,
				"duration", duration,
			)
		}
		return resp, err
	}
}

// TokenProvider is a function that returns an auth token
type TokenProvider func(ctx context.Context) (string, error)

// AuthHeaderInterceptor adds an auth header to all outgoing requests
func AuthHeaderInterceptor(tokenProvider TokenProvider) ClientInterceptor {
	return func(ctx context.Context, req *CmdReq, invoker ClientInvoker) (*CmdResp, error) {
		token, err := tokenProvider(ctx)
		if err != nil {
			return nil, err
		}
		if token != "" {
			req.WithAuth(token)
		}
		return invoker(ctx, req)
	}
}

// StaticAuthHeaderInterceptor adds a static auth header to all outgoing requests
func StaticAuthHeaderInterceptor(token string) ClientInterceptor {
	return func(ctx context.Context, req *CmdReq, invoker ClientInvoker) (*CmdResp, error) {
		if token != "" {
			req.WithAuth(token)
		}
		return invoker(ctx, req)
	}
}

// HeaderInterceptor adds custom headers to all outgoing requests
func HeaderInterceptor(headers map[string]string) ClientInterceptor {
	return func(ctx context.Context, req *CmdReq, invoker ClientInvoker) (*CmdResp, error) {
		for k, v := range headers {
			req.WithHeader(k, v)
		}
		return invoker(ctx, req)
	}
}

// RetryInterceptor retries failed requests up to maxRetries times
// It only retries on transport errors, not on application-level errors
func RetryInterceptor(maxRetries int, delay time.Duration) ClientInterceptor {
	return func(ctx context.Context, req *CmdReq, invoker ClientInvoker) (*CmdResp, error) {
		var resp *CmdResp
		var err error

		for attempt := 0; attempt <= maxRetries; attempt++ {
			resp, err = invoker(ctx, req)
			if err == nil {
				return resp, nil
			}

			// Don't retry if context is cancelled
			if ctx.Err() != nil {
				return resp, err
			}

			// Don't retry on last attempt
			if attempt < maxRetries {
				slog.Warn("client.retry",
					"cmd", req.Cmd,
					"attempt", attempt+1,
					"maxRetries", maxRetries,
					"error", err.Error(),
				)
				time.Sleep(delay)
			}
		}
		return resp, err
	}
}

// ClientRequestIDInterceptor adds a request ID to outgoing requests if not present
func ClientRequestIDInterceptor(generator func() string) ClientInterceptor {
	return func(ctx context.Context, req *CmdReq, invoker ClientInvoker) (*CmdResp, error) {
		if req.Headers["request-id"] == "" && generator != nil {
			req.WithHeader("request-id", generator())
		}
		return invoker(ctx, req)
	}
}
