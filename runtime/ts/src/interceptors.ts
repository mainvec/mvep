/**
 * Built-in interceptors for MVP client and server
 */

import type { CmdReq, CmdResp, MvpContext } from './envelope';
import { newCmdRespError, hasError, contextWithUser } from './envelope';
import type { CmdHandler, CmdInterceptor, ClientInvoker, ClientInterceptor } from './middleware';

// =============================================================================
// Server-side Interceptors
// =============================================================================

/**
 * Logger interface for interceptors
 */
export interface Logger {
  info(message: string, ...args: unknown[]): void;
  error(message: string, ...args: unknown[]): void;
  warn(message: string, ...args: unknown[]): void;
  debug(message: string, ...args: unknown[]): void;
}

/**
 * Default console logger
 */
export const consoleLogger: Logger = {
  info: (msg, ...args) => console.log(`[INFO] ${msg}`, ...args),
  error: (msg, ...args) => console.error(`[ERROR] ${msg}`, ...args),
  warn: (msg, ...args) => console.warn(`[WARN] ${msg}`, ...args),
  debug: (msg, ...args) => console.debug(`[DEBUG] ${msg}`, ...args),
};

/**
 * LoggingInterceptor logs command execution with timing information
 */
export function loggingInterceptor(logger: Logger = consoleLogger): CmdInterceptor {
  return async (ctx: MvpContext, req: CmdReq, next: CmdHandler): Promise<CmdResp> => {
    const start = Date.now();
    logger.info('cmd.start', { cmd: req.cmd });

    const resp = await next(ctx, req);

    const duration = Date.now() - start;
    if (hasError(resp)) {
      logger.error('cmd.end', {
        cmd: req.cmd,
        duration: `${duration}ms`,
        errorCode: resp.error?.code,
        errorMsg: resp.error?.message,
      });
    } else {
      logger.info('cmd.end', { cmd: req.cmd, duration: `${duration}ms` });
    }

    return resp;
  };
}

/**
 * TokenValidator interface for validating authentication tokens
 */
export interface TokenValidator {
  /**
   * Validates the token and returns user info
   * Throws an error if the token is invalid
   */
  validate(token: string): Promise<unknown>;
}

/**
 * AuthInterceptor validates the "auth" header using the provided validator
 * On success, it stores the user info in the context accessible via ctx.user
 */
export function authInterceptor(validator: TokenValidator): CmdInterceptor {
  return async (ctx: MvpContext, req: CmdReq, next: CmdHandler): Promise<CmdResp> => {
    const token = req.headers['auth'];
    if (!token) {
      return newCmdRespError('unauthorized', 'missing auth token');
    }

    try {
      const user = await validator.validate(token);
      ctx = contextWithUser(ctx, user);
      return next(ctx, req);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'invalid token';
      return newCmdRespError('unauthorized', message);
    }
  };
}

/**
 * RecoveryInterceptor recovers from errors in command handlers
 * and returns an error response instead of throwing
 */
export function recoveryInterceptor(logger: Logger = consoleLogger): CmdInterceptor {
  return async (ctx: MvpContext, req: CmdReq, next: CmdHandler): Promise<CmdResp> => {
    try {
      return await next(ctx, req);
    } catch (err) {
      logger.error('cmd.panic', {
        cmd: req.cmd,
        error: err instanceof Error ? err.message : String(err),
      });
      return newCmdRespError('internal_error', 'internal server error');
    }
  };
}

/**
 * RequestIDInterceptor ensures every request has a request ID
 * If not present in headers, generates one and adds it to both request and response
 */
export function requestIdInterceptor(generator: () => string): CmdInterceptor {
  return async (ctx: MvpContext, req: CmdReq, next: CmdHandler): Promise<CmdResp> => {
    let requestId = req.headers['request-id'];
    if (!requestId && generator) {
      requestId = generator();
      req.headers['request-id'] = requestId;
    }

    const resp = await next(ctx, req);

    // Echo request ID in response
    if (requestId) {
      resp.headers['request-id'] = requestId;
    }

    return resp;
  };
}

// =============================================================================
// Client-side Interceptors
// =============================================================================

/**
 * ClientLoggingInterceptor logs client requests and responses with timing
 */
export function clientLoggingInterceptor(logger: Logger = consoleLogger): ClientInterceptor {
  return async (ctx: MvpContext, req: CmdReq, invoker: ClientInvoker): Promise<CmdResp> => {
    const start = Date.now();
    logger.info('client.request', { cmd: req.cmd });

    try {
      const resp = await invoker(ctx, req);
      const duration = Date.now() - start;

      if (hasError(resp)) {
        logger.error('client.response', {
          cmd: req.cmd,
          duration: `${duration}ms`,
          errorCode: resp.error?.code,
          errorMsg: resp.error?.message,
        });
      } else {
        logger.info('client.response', { cmd: req.cmd, duration: `${duration}ms` });
      }

      return resp;
    } catch (err) {
      const duration = Date.now() - start;
      logger.error('client.response', {
        cmd: req.cmd,
        duration: `${duration}ms`,
        error: err instanceof Error ? err.message : String(err),
      });
      throw err;
    }
  };
}

/**
 * TokenProvider is a function that returns an auth token
 */
export type TokenProvider = () => Promise<string> | string;

/**
 * AuthHeaderInterceptor adds an auth header to all outgoing requests
 */
export function authHeaderInterceptor(tokenProvider: TokenProvider): ClientInterceptor {
  return async (ctx: MvpContext, req: CmdReq, invoker: ClientInvoker): Promise<CmdResp> => {
    const token = await tokenProvider();
    if (token) {
      req.headers['auth'] = token;
    }
    return invoker(ctx, req);
  };
}

/**
 * StaticAuthHeaderInterceptor adds a static auth header to all outgoing requests
 */
export function staticAuthHeaderInterceptor(token: string): ClientInterceptor {
  return async (ctx: MvpContext, req: CmdReq, invoker: ClientInvoker): Promise<CmdResp> => {
    if (token) {
      req.headers['auth'] = token;
    }
    return invoker(ctx, req);
  };
}

/**
 * HeaderInterceptor adds custom headers to all outgoing requests
 */
export function headerInterceptor(headers: Record<string, string>): ClientInterceptor {
  return async (ctx: MvpContext, req: CmdReq, invoker: ClientInvoker): Promise<CmdResp> => {
    for (const [key, value] of Object.entries(headers)) {
      req.headers[key] = value;
    }
    return invoker(ctx, req);
  };
}

/**
 * Sleep utility for retry interceptor
 */
function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * RetryInterceptor retries failed requests up to maxRetries times
 * It only retries on transport errors, not on application-level errors
 */
export function retryInterceptor(maxRetries: number, delayMs: number): ClientInterceptor {
  return async (ctx: MvpContext, req: CmdReq, invoker: ClientInvoker): Promise<CmdResp> => {
    let lastError: Error | undefined;

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        return await invoker(ctx, req);
      } catch (err) {
        lastError = err instanceof Error ? err : new Error(String(err));

        // Don't retry on last attempt
        if (attempt < maxRetries) {
          consoleLogger.warn('client.retry', {
            cmd: req.cmd,
            attempt: attempt + 1,
            maxRetries,
            error: lastError.message,
          });
          await sleep(delayMs);
        }
      }
    }

    throw lastError;
  };
}

/**
 * ClientRequestIDInterceptor adds a request ID to outgoing requests if not present
 */
export function clientRequestIdInterceptor(generator: () => string): ClientInterceptor {
  return async (ctx: MvpContext, req: CmdReq, invoker: ClientInvoker): Promise<CmdResp> => {
    if (!req.headers['request-id'] && generator) {
      req.headers['request-id'] = generator();
    }
    return invoker(ctx, req);
  };
}

/**
 * TimeoutInterceptor adds a timeout to requests
 */
export function timeoutInterceptor(timeoutMs: number): ClientInterceptor {
  return async (ctx: MvpContext, req: CmdReq, invoker: ClientInvoker): Promise<CmdResp> => {
    const timeoutPromise = new Promise<never>((_, reject) => {
      setTimeout(() => reject(new Error(`Request timeout after ${timeoutMs}ms`)), timeoutMs);
    });

    return Promise.race([invoker(ctx, req), timeoutPromise]);
  };
}

/**
 * Simple UUID v4 generator for request IDs
 */
export function generateRequestId(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}
