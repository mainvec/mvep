/**
 * Envelope types for MVP command request/response handling
 */

/** HeaderPrefix is the prefix for custom MVP headers in HTTP transport */
export const HEADER_PREFIX = 'x-mvp-';

/**
 * ErrorInfo provides structured error information
 */
export interface ErrorInfo {
  code: string;
  message: string;
}

/**
 * CmdReq represents a command request with headers
 */
export interface CmdReq {
  /** Cmd is the command name */
  cmd: string;
  /** Headers contains request metadata (auth tokens, trace IDs, etc.) */
  headers: Record<string, string>;
  /** Payload is the encoded command bytes/data */
  payload: unknown;
}

/**
 * CmdResp represents a command response with headers
 */
export interface CmdResp {
  /** Headers contains response metadata (new tokens, rate limits, etc.) */
  headers: Record<string, string>;
  /** Payload is the encoded result */
  payload: unknown;
  /** Error contains error information if the command failed */
  error?: ErrorInfo;
}

/**
 * Creates a new CmdReq with initialized headers map
 */
export function newCmdReq(cmd: string, payload?: unknown): CmdReq {
  return {
    cmd,
    headers: {},
    payload,
  };
}

/**
 * Creates a new CmdResp with initialized headers map
 */
export function newCmdResp(payload?: unknown): CmdResp {
  return {
    headers: {},
    payload,
  };
}

/**
 * Creates a new CmdResp with an error
 */
export function newCmdRespError(code: string, message: string): CmdResp {
  return {
    headers: {},
    payload: undefined,
    error: { code, message },
  };
}

/**
 * Helper class for building CmdReq with method chaining
 */
export class CmdReqBuilder {
  private req: CmdReq;

  constructor(cmd: string, payload?: unknown) {
    this.req = newCmdReq(cmd, payload);
  }

  /**
   * Adds a header to the request
   */
  withHeader(key: string, value: string): this {
    this.req.headers[key] = value;
    return this;
  }

  /**
   * Adds an authorization header
   */
  withAuth(token: string): this {
    return this.withHeader('auth', token);
  }

  /**
   * Sets the payload
   */
  withPayload(payload: unknown): this {
    this.req.payload = payload;
    return this;
  }

  /**
   * Builds and returns the CmdReq
   */
  build(): CmdReq {
    return this.req;
  }
}

/**
 * Helper class for building CmdResp with method chaining
 */
export class CmdRespBuilder {
  private resp: CmdResp;

  constructor(payload?: unknown) {
    this.resp = newCmdResp(payload);
  }

  /**
   * Adds a header to the response
   */
  withHeader(key: string, value: string): this {
    this.resp.headers[key] = value;
    return this;
  }

  /**
   * Sets an error on the response
   */
  withError(code: string, message: string): this {
    this.resp.error = { code, message };
    return this;
  }

  /**
   * Builds and returns the CmdResp
   */
  build(): CmdResp {
    return this.resp;
  }
}

/**
 * Checks if a response has an error
 */
export function hasError(resp: CmdResp): boolean {
  return resp.error !== undefined && resp.error !== null;
}

/**
 * Gets a header value from the response
 */
export function getHeader(resp: CmdResp, key: string): string | undefined {
  return resp.headers[key];
}

/**
 * Context type for passing request/response through async operations
 */
export interface MvpContext {
  /** Request being processed */
  request?: CmdReq;
  /** Response being built */
  response?: CmdResp;
  /** User data from authentication */
  user?: unknown;
  /** Request ID */
  requestId?: string;
  /** Additional context values */
  values: Record<string, unknown>;
}

/**
 * Creates a new empty context
 */
export function newContext(): MvpContext {
  return {
    values: {},
  };
}

/**
 * Creates a context with a request
 */
export function contextWithRequest(ctx: MvpContext, req: CmdReq): MvpContext {
  return {
    ...ctx,
    request: req,
  };
}

/**
 * Creates a context with a response
 */
export function contextWithResponse(ctx: MvpContext, resp: CmdResp): MvpContext {
  return {
    ...ctx,
    response: resp,
  };
}

/**
 * Creates a context with user data
 */
export function contextWithUser(ctx: MvpContext, user: unknown): MvpContext {
  return {
    ...ctx,
    user,
  };
}

/**
 * Gets the request header from context
 */
export function getRequestHeader(ctx: MvpContext, key: string): string | undefined {
  return ctx.request?.headers[key];
}

/**
 * Sets a response header in context (mutates the context response)
 */
export function setResponseHeader(ctx: MvpContext, key: string, value: string): MvpContext {
  if (!ctx.response) {
    ctx.response = newCmdResp();
  }
  ctx.response.headers[key] = value;
  return ctx;
}
