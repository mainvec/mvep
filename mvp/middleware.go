package mvp

import "context"

// CmdHandler is the function signature for handling commands
type CmdHandler func(ctx context.Context, req *CmdReq) *CmdResp

// CmdInterceptor wraps a CmdHandler with additional logic
// Interceptors can modify the request, short-circuit execution, or modify the response
type CmdInterceptor func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp

// Chain combines multiple interceptors into a single interceptor
// Interceptors are executed in the order they are provided
func Chain(interceptors ...CmdInterceptor) CmdInterceptor {
	if len(interceptors) == 0 {
		return nil
	}
	if len(interceptors) == 1 {
		return interceptors[0]
	}

	return func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		// Build chain from right to left so execution is left to right
		handler := next
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			currentHandler := handler
			handler = func(ctx context.Context, req *CmdReq) *CmdResp {
				return interceptor(ctx, req, currentHandler)
			}
		}
		return handler(ctx, req)
	}
}

// SkipCommands wraps an interceptor to skip certain commands
// Commands in the skip list will bypass this interceptor entirely
func SkipCommands(interceptor CmdInterceptor, commands ...string) CmdInterceptor {
	if interceptor == nil {
		return nil
	}
	skipSet := make(map[string]bool, len(commands))
	for _, cmd := range commands {
		skipSet[cmd] = true
	}

	return func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		// Skip this interceptor for commands in the skip list
		if skipSet[req.Cmd] {
			return next(ctx, req)
		}
		return interceptor(ctx, req, next)
	}
}

// OnlyCommands wraps an interceptor to only apply to certain commands
// Only commands in the list will go through this interceptor
func OnlyCommands(interceptor CmdInterceptor, commands ...string) CmdInterceptor {
	if interceptor == nil {
		return nil
	}
	onlySet := make(map[string]bool, len(commands))
	for _, cmd := range commands {
		onlySet[cmd] = true
	}

	return func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		// Only apply this interceptor for commands in the list
		if !onlySet[req.Cmd] {
			return next(ctx, req)
		}
		return interceptor(ctx, req, next)
	}
}

// =============================================================================
// Client-side Interceptors
// =============================================================================

// ClientInvoker calls the next step in the client interceptor chain
type ClientInvoker func(ctx context.Context, req *CmdReq) (*CmdResp, error)

// ClientInterceptor wraps a client call with additional logic
// Interceptors can modify the request, add headers, retry, log, etc.
type ClientInterceptor func(ctx context.Context, req *CmdReq, invoker ClientInvoker) (*CmdResp, error)

// ChainClient combines multiple client interceptors into a single interceptor
// Interceptors are executed in the order they are provided
func ChainClient(interceptors ...ClientInterceptor) ClientInterceptor {
	if len(interceptors) == 0 {
		return nil
	}
	if len(interceptors) == 1 {
		return interceptors[0]
	}

	return func(ctx context.Context, req *CmdReq, invoker ClientInvoker) (*CmdResp, error) {
		// Build chain from right to left so execution is left to right
		currentInvoker := invoker
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			nextInvoker := currentInvoker
			currentInvoker = func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
				return interceptor(ctx, req, nextInvoker)
			}
		}
		return currentInvoker(ctx, req)
	}
}

// SkipCommandsClient wraps a client interceptor to skip certain commands
func SkipCommandsClient(interceptor ClientInterceptor, commands ...string) ClientInterceptor {
	if interceptor == nil {
		return nil
	}
	skipSet := make(map[string]bool, len(commands))
	for _, cmd := range commands {
		skipSet[cmd] = true
	}

	return func(ctx context.Context, req *CmdReq, invoker ClientInvoker) (*CmdResp, error) {
		if skipSet[req.Cmd] {
			return invoker(ctx, req)
		}
		return interceptor(ctx, req, invoker)
	}
}
