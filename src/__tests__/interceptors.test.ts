import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  loggingInterceptor,
  authInterceptor,
  recoveryInterceptor,
  requestIdInterceptor,
  clientLoggingInterceptor,
  authHeaderInterceptor,
  staticAuthHeaderInterceptor,
  headerInterceptor,
  retryInterceptor,
  clientRequestIdInterceptor,
  timeoutInterceptor,
  generateRequestId,
  consoleLogger,
} from '../interceptors';
import { newCmdReq, newCmdResp, newCmdRespError, newContext, hasError } from '../envelope';
import type { Logger, TokenValidator } from '../interceptors';

describe('interceptors', () => {
  const mockLogger: Logger = {
    info: vi.fn(),
    error: vi.fn(),
    warn: vi.fn(),
    debug: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('loggingInterceptor', () => {
    it('logs command start and end', async () => {
      const interceptor = loggingInterceptor(mockLogger);
      const handler = vi.fn().mockResolvedValue(newCmdResp({ result: 'ok' }));
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      await interceptor(ctx, req, handler);

      expect(mockLogger.info).toHaveBeenCalledWith('cmd.start', expect.objectContaining({ cmd: 'TestCmd' }));
      expect(mockLogger.info).toHaveBeenCalledWith('cmd.end', expect.objectContaining({ cmd: 'TestCmd' }));
    });

    it('logs errors', async () => {
      const interceptor = loggingInterceptor(mockLogger);
      const handler = vi.fn().mockResolvedValue(newCmdRespError('error', 'something failed'));
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      await interceptor(ctx, req, handler);

      expect(mockLogger.error).toHaveBeenCalledWith('cmd.end', expect.objectContaining({
        cmd: 'TestCmd',
        errorCode: 'error',
        errorMsg: 'something failed',
      }));
    });
  });

  describe('authInterceptor', () => {
    it('returns error when auth header is missing', async () => {
      const validator: TokenValidator = { validate: vi.fn() };
      const interceptor = authInterceptor(validator);
      const handler = vi.fn().mockResolvedValue(newCmdResp());
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      const resp = await interceptor(ctx, req, handler);

      expect(hasError(resp)).toBe(true);
      expect(resp.error?.code).toBe('unauthorized');
      expect(resp.error?.message).toBe('missing auth token');
      expect(handler).not.toHaveBeenCalled();
    });

    it('validates token and continues on success', async () => {
      const user = { id: 1, name: 'Test User' };
      const validator: TokenValidator = {
        validate: vi.fn().mockResolvedValue(user),
      };
      const interceptor = authInterceptor(validator);
      const handler = vi.fn().mockResolvedValue(newCmdResp({ result: 'ok' }));
      const ctx = newContext();
      const req = newCmdReq('TestCmd');
      req.headers['auth'] = 'valid-token';

      const resp = await interceptor(ctx, req, handler);

      expect(validator.validate).toHaveBeenCalledWith('valid-token');
      expect(handler).toHaveBeenCalledWith(
        expect.objectContaining({ user }),
        req
      );
      expect(hasError(resp)).toBe(false);
    });

    it('returns error on validation failure', async () => {
      const validator: TokenValidator = {
        validate: vi.fn().mockRejectedValue(new Error('invalid token')),
      };
      const interceptor = authInterceptor(validator);
      const handler = vi.fn();
      const ctx = newContext();
      const req = newCmdReq('TestCmd');
      req.headers['auth'] = 'invalid-token';

      const resp = await interceptor(ctx, req, handler);

      expect(hasError(resp)).toBe(true);
      expect(resp.error?.code).toBe('unauthorized');
      expect(resp.error?.message).toBe('invalid token');
    });
  });

  describe('recoveryInterceptor', () => {
    it('catches errors and returns error response', async () => {
      const interceptor = recoveryInterceptor(mockLogger);
      const handler = vi.fn().mockRejectedValue(new Error('unexpected error'));
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      const resp = await interceptor(ctx, req, handler);

      expect(hasError(resp)).toBe(true);
      expect(resp.error?.code).toBe('internal_error');
      expect(mockLogger.error).toHaveBeenCalled();
    });
  });

  describe('requestIdInterceptor', () => {
    it('generates request ID if not present', async () => {
      const generator = vi.fn().mockReturnValue('generated-id');
      const interceptor = requestIdInterceptor(generator);
      const handler = vi.fn().mockResolvedValue(newCmdResp());
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      const resp = await interceptor(ctx, req, handler);

      expect(generator).toHaveBeenCalled();
      expect(req.headers['request-id']).toBe('generated-id');
      expect(resp.headers['request-id']).toBe('generated-id');
    });

    it('uses existing request ID', async () => {
      const generator = vi.fn();
      const interceptor = requestIdInterceptor(generator);
      const handler = vi.fn().mockResolvedValue(newCmdResp());
      const ctx = newContext();
      const req = newCmdReq('TestCmd');
      req.headers['request-id'] = 'existing-id';

      const resp = await interceptor(ctx, req, handler);

      expect(generator).not.toHaveBeenCalled();
      expect(resp.headers['request-id']).toBe('existing-id');
    });
  });

  describe('clientLoggingInterceptor', () => {
    it('logs client requests and responses', async () => {
      const interceptor = clientLoggingInterceptor(mockLogger);
      const invoker = vi.fn().mockResolvedValue(newCmdResp({ result: 'ok' }));
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      await interceptor(ctx, req, invoker);

      expect(mockLogger.info).toHaveBeenCalledWith('client.request', expect.objectContaining({ cmd: 'TestCmd' }));
      expect(mockLogger.info).toHaveBeenCalledWith('client.response', expect.objectContaining({ cmd: 'TestCmd' }));
    });
  });

  describe('authHeaderInterceptor', () => {
    it('adds auth header from provider', async () => {
      const tokenProvider = vi.fn().mockResolvedValue('my-token');
      const interceptor = authHeaderInterceptor(tokenProvider);
      const invoker = vi.fn().mockResolvedValue(newCmdResp());
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      await interceptor(ctx, req, invoker);

      expect(tokenProvider).toHaveBeenCalled();
      expect(req.headers['auth']).toBe('my-token');
    });
  });

  describe('staticAuthHeaderInterceptor', () => {
    it('adds static auth header', async () => {
      const interceptor = staticAuthHeaderInterceptor('static-token');
      const invoker = vi.fn().mockResolvedValue(newCmdResp());
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      await interceptor(ctx, req, invoker);

      expect(req.headers['auth']).toBe('static-token');
    });
  });

  describe('headerInterceptor', () => {
    it('adds custom headers', async () => {
      const interceptor = headerInterceptor({
        'x-custom': 'value1',
        'x-another': 'value2',
      });
      const invoker = vi.fn().mockResolvedValue(newCmdResp());
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      await interceptor(ctx, req, invoker);

      expect(req.headers['x-custom']).toBe('value1');
      expect(req.headers['x-another']).toBe('value2');
    });
  });

  describe('retryInterceptor', () => {
    it('retries on failure', async () => {
      vi.useFakeTimers();
      const interceptor = retryInterceptor(2, 100);
      const invoker = vi.fn()
        .mockRejectedValueOnce(new Error('fail 1'))
        .mockRejectedValueOnce(new Error('fail 2'))
        .mockResolvedValue(newCmdResp({ result: 'ok' }));
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      const promise = interceptor(ctx, req, invoker);
      
      // Fast-forward through retries
      await vi.runAllTimersAsync();
      
      const resp = await promise;

      expect(invoker).toHaveBeenCalledTimes(3);
      expect(hasError(resp)).toBe(false);
      
      vi.useRealTimers();
    });

    it('throws after max retries', async () => {
      vi.useFakeTimers();
      const interceptor = retryInterceptor(2, 100);
      const invoker = vi.fn().mockRejectedValue(new Error('persistent failure'));
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      const promise = interceptor(ctx, req, invoker);
      
      // Handle the promise rejection before running timers
      const resultPromise = promise.catch(e => e);
      await vi.runAllTimersAsync();
      
      const error = await resultPromise;
      expect(error).toBeInstanceOf(Error);
      expect(error.message).toBe('persistent failure');
      expect(invoker).toHaveBeenCalledTimes(3);
      
      vi.useRealTimers();
    });
  });

  describe('clientRequestIdInterceptor', () => {
    it('generates request ID for client requests', async () => {
      const generator = vi.fn().mockReturnValue('client-request-id');
      const interceptor = clientRequestIdInterceptor(generator);
      const invoker = vi.fn().mockResolvedValue(newCmdResp());
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      await interceptor(ctx, req, invoker);

      expect(req.headers['request-id']).toBe('client-request-id');
    });
  });

  describe('timeoutInterceptor', () => {
    it('times out long requests', async () => {
      vi.useFakeTimers();
      const interceptor = timeoutInterceptor(1000);
      const invoker = vi.fn().mockImplementation(() => new Promise(() => {})); // Never resolves
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      const promise = interceptor(ctx, req, invoker);
      vi.advanceTimersByTime(1001);

      await expect(promise).rejects.toThrow('Request timeout after 1000ms');
      
      vi.useRealTimers();
    });
  });

  describe('generateRequestId', () => {
    it('generates UUID-like strings', () => {
      const id1 = generateRequestId();
      const id2 = generateRequestId();

      expect(id1).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
      expect(id1).not.toBe(id2);
    });
  });
});
