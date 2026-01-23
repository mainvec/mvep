/**
 * Middleware types and utilities for MVP command handling
 */

import type { CmdReq, CmdResp, MvpContext } from './envelope';

/**
 * CmdHandler is the function signature for handling commands (server-side)
 */
export type CmdHandler = (ctx: MvpContext, req: CmdReq) => CmdResp | Promise<CmdResp>;

/**
 * CmdInterceptor wraps a CmdHandler with additional logic (server-side)
 * Interceptors can modify the request, short-circuit execution, or modify the response
 */
export type CmdInterceptor = (
  ctx: MvpContext,
  req: CmdReq,
  next: CmdHandler
) => CmdResp | Promise<CmdResp>;

/**
 * ClientInvoker calls the next step in the client interceptor chain
 */
export type ClientInvoker = (ctx: MvpContext, req: CmdReq) => Promise<CmdResp>;

/**
 * ClientInterceptor wraps a client call with additional logic
 * Interceptors can modify the request, add headers, retry, log, etc.
 */
export type ClientInterceptor = (
  ctx: MvpContext,
  req: CmdReq,
  invoker: ClientInvoker
) => Promise<CmdResp>;

/**
 * Chains multiple server-side interceptors into a single interceptor
 * Interceptors are executed in the order they are provided
 */
export function chain(...interceptors: CmdInterceptor[]): CmdInterceptor | null {
  if (interceptors.length === 0) {
    return null;
  }
  if (interceptors.length === 1) {
    return interceptors[0];
  }

  return async (ctx: MvpContext, req: CmdReq, next: CmdHandler): Promise<CmdResp> => {
    // Build chain from right to left so execution is left to right
    let handler: CmdHandler = next;
    
    for (let i = interceptors.length - 1; i >= 0; i--) {
      const interceptor = interceptors[i];
      const currentHandler = handler;
      handler = (ctx: MvpContext, req: CmdReq) => interceptor(ctx, req, currentHandler);
    }
    
    return handler(ctx, req);
  };
}

/**
 * Chains multiple client-side interceptors into a single interceptor
 * Interceptors are executed in the order they are provided
 */
export function chainClient(...interceptors: ClientInterceptor[]): ClientInterceptor | null {
  if (interceptors.length === 0) {
    return null;
  }
  if (interceptors.length === 1) {
    return interceptors[0];
  }

  return async (ctx: MvpContext, req: CmdReq, invoker: ClientInvoker): Promise<CmdResp> => {
    // Build chain from right to left so execution is left to right
    let currentInvoker: ClientInvoker = invoker;
    
    for (let i = interceptors.length - 1; i >= 0; i--) {
      const interceptor = interceptors[i];
      const nextInvoker = currentInvoker;
      currentInvoker = (ctx: MvpContext, req: CmdReq) => interceptor(ctx, req, nextInvoker);
    }
    
    return currentInvoker(ctx, req);
  };
}

/**
 * Wraps a server interceptor to skip certain commands
 * Commands in the skip list will bypass this interceptor entirely
 */
export function skipCommands(
  interceptor: CmdInterceptor,
  ...commands: string[]
): CmdInterceptor {
  const skipSet = new Set(commands);

  return async (ctx: MvpContext, req: CmdReq, next: CmdHandler): Promise<CmdResp> => {
    // Skip this interceptor for commands in the skip list
    if (skipSet.has(req.cmd)) {
      return next(ctx, req);
    }
    return interceptor(ctx, req, next);
  };
}

/**
 * Wraps a server interceptor to only apply to certain commands
 * Only commands in the list will go through this interceptor
 */
export function onlyCommands(
  interceptor: CmdInterceptor,
  ...commands: string[]
): CmdInterceptor {
  const onlySet = new Set(commands);

  return async (ctx: MvpContext, req: CmdReq, next: CmdHandler): Promise<CmdResp> => {
    // Only apply this interceptor for commands in the list
    if (!onlySet.has(req.cmd)) {
      return next(ctx, req);
    }
    return interceptor(ctx, req, next);
  };
}

/**
 * Wraps a client interceptor to skip certain commands
 */
export function skipCommandsClient(
  interceptor: ClientInterceptor,
  ...commands: string[]
): ClientInterceptor {
  const skipSet = new Set(commands);

  return async (ctx: MvpContext, req: CmdReq, invoker: ClientInvoker): Promise<CmdResp> => {
    if (skipSet.has(req.cmd)) {
      return invoker(ctx, req);
    }
    return interceptor(ctx, req, invoker);
  };
}

/**
 * Wraps a client interceptor to only apply to certain commands
 */
export function onlyCommandsClient(
  interceptor: ClientInterceptor,
  ...commands: string[]
): ClientInterceptor {
  const onlySet = new Set(commands);

  return async (ctx: MvpContext, req: CmdReq, invoker: ClientInvoker): Promise<CmdResp> => {
    if (!onlySet.has(req.cmd)) {
      return invoker(ctx, req);
    }
    return interceptor(ctx, req, invoker);
  };
}
