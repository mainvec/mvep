import { describe, it, expect, vi } from 'vitest';
import {
  chain,
  chainClient,
  skipCommands,
  onlyCommands,
  skipCommandsClient,
  onlyCommandsClient,
} from '../middleware';
import type { CmdInterceptor, ClientInterceptor } from '../middleware';
import { newCmdReq, newCmdResp, newContext } from '../envelope';
import type { MvpContext, CmdReq, CmdResp } from '../envelope';

describe('middleware', () => {
  describe('chain', () => {
    it('returns null for empty array', () => {
      expect(chain()).toBeNull();
    });

    it('returns single interceptor unchanged', () => {
      const interceptor: CmdInterceptor = async (ctx, req, next) => next(ctx, req);
      expect(chain(interceptor)).toBe(interceptor);
    });

    it('chains multiple interceptors in order', async () => {
      const order: string[] = [];

      const first: CmdInterceptor = async (ctx, req, next) => {
        order.push('first-before');
        const resp = await next(ctx, req);
        order.push('first-after');
        return resp;
      };

      const second: CmdInterceptor = async (ctx, req, next) => {
        order.push('second-before');
        const resp = await next(ctx, req);
        order.push('second-after');
        return resp;
      };

      const handler = async (ctx: MvpContext, req: CmdReq): Promise<CmdResp> => {
        order.push('handler');
        return newCmdResp({ result: 'ok' });
      };

      const chained = chain(first, second)!;
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      await chained(ctx, req, handler);

      expect(order).toEqual([
        'first-before',
        'second-before',
        'handler',
        'second-after',
        'first-after',
      ]);
    });
  });

  describe('chainClient', () => {
    it('returns null for empty array', () => {
      expect(chainClient()).toBeNull();
    });

    it('returns single interceptor unchanged', () => {
      const interceptor: ClientInterceptor = async (ctx, req, invoker) => invoker(ctx, req);
      expect(chainClient(interceptor)).toBe(interceptor);
    });

    it('chains multiple client interceptors in order', async () => {
      const order: string[] = [];

      const first: ClientInterceptor = async (ctx, req, invoker) => {
        order.push('first-before');
        const resp = await invoker(ctx, req);
        order.push('first-after');
        return resp;
      };

      const second: ClientInterceptor = async (ctx, req, invoker) => {
        order.push('second-before');
        const resp = await invoker(ctx, req);
        order.push('second-after');
        return resp;
      };

      const invoker = async (ctx: MvpContext, req: CmdReq): Promise<CmdResp> => {
        order.push('invoker');
        return newCmdResp({ result: 'ok' });
      };

      const chained = chainClient(first, second)!;
      const ctx = newContext();
      const req = newCmdReq('TestCmd');

      await chained(ctx, req, invoker);

      expect(order).toEqual([
        'first-before',
        'second-before',
        'invoker',
        'second-after',
        'first-after',
      ]);
    });
  });

  describe('skipCommands', () => {
    it('skips specified commands', async () => {
      const interceptorCalled = vi.fn();
      
      const interceptor: CmdInterceptor = async (ctx, req, next) => {
        interceptorCalled();
        return next(ctx, req);
      };

      const wrapped = skipCommands(interceptor, 'SkipMe', 'AlsoSkip');
      const handler = async () => newCmdResp();
      const ctx = newContext();

      // Should skip
      await wrapped(ctx, newCmdReq('SkipMe'), handler);
      expect(interceptorCalled).not.toHaveBeenCalled();

      // Should not skip
      await wrapped(ctx, newCmdReq('DontSkip'), handler);
      expect(interceptorCalled).toHaveBeenCalledTimes(1);
    });
  });

  describe('onlyCommands', () => {
    it('only applies to specified commands', async () => {
      const interceptorCalled = vi.fn();
      
      const interceptor: CmdInterceptor = async (ctx, req, next) => {
        interceptorCalled();
        return next(ctx, req);
      };

      const wrapped = onlyCommands(interceptor, 'ApplyHere', 'AndHere');
      const handler = async () => newCmdResp();
      const ctx = newContext();

      // Should apply
      await wrapped(ctx, newCmdReq('ApplyHere'), handler);
      expect(interceptorCalled).toHaveBeenCalledTimes(1);

      // Should not apply
      await wrapped(ctx, newCmdReq('NotHere'), handler);
      expect(interceptorCalled).toHaveBeenCalledTimes(1); // Still 1
    });
  });

  describe('skipCommandsClient', () => {
    it('skips specified commands for client interceptor', async () => {
      const interceptorCalled = vi.fn();
      
      const interceptor: ClientInterceptor = async (ctx, req, invoker) => {
        interceptorCalled();
        return invoker(ctx, req);
      };

      const wrapped = skipCommandsClient(interceptor, 'SkipMe');
      const invoker = async () => newCmdResp();
      const ctx = newContext();

      await wrapped(ctx, newCmdReq('SkipMe'), invoker);
      expect(interceptorCalled).not.toHaveBeenCalled();

      await wrapped(ctx, newCmdReq('DontSkip'), invoker);
      expect(interceptorCalled).toHaveBeenCalledTimes(1);
    });
  });

  describe('onlyCommandsClient', () => {
    it('only applies to specified commands for client interceptor', async () => {
      const interceptorCalled = vi.fn();
      
      const interceptor: ClientInterceptor = async (ctx, req, invoker) => {
        interceptorCalled();
        return invoker(ctx, req);
      };

      const wrapped = onlyCommandsClient(interceptor, 'ApplyHere');
      const invoker = async () => newCmdResp();
      const ctx = newContext();

      await wrapped(ctx, newCmdReq('ApplyHere'), invoker);
      expect(interceptorCalled).toHaveBeenCalledTimes(1);

      await wrapped(ctx, newCmdReq('NotHere'), invoker);
      expect(interceptorCalled).toHaveBeenCalledTimes(1); // Still 1
    });
  });
});
