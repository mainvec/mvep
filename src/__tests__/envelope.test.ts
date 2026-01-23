import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  newCmdReq,
  newCmdResp,
  newCmdRespError,
  CmdReqBuilder,
  CmdRespBuilder,
  hasError,
  getHeader,
  newContext,
  contextWithRequest,
  contextWithResponse,
  contextWithUser,
  getRequestHeader,
  setResponseHeader,
  HEADER_PREFIX,
} from '../envelope';

describe('envelope', () => {
  describe('newCmdReq', () => {
    it('creates a new request with command name', () => {
      const req = newCmdReq('TestCmd', { foo: 'bar' });
      
      expect(req.cmd).toBe('TestCmd');
      expect(req.payload).toEqual({ foo: 'bar' });
      expect(req.headers).toEqual({});
    });

    it('creates a request without payload', () => {
      const req = newCmdReq('TestCmd');
      
      expect(req.cmd).toBe('TestCmd');
      expect(req.payload).toBeUndefined();
    });
  });

  describe('newCmdResp', () => {
    it('creates a new response with payload', () => {
      const resp = newCmdResp({ result: 'ok' });
      
      expect(resp.payload).toEqual({ result: 'ok' });
      expect(resp.headers).toEqual({});
      expect(resp.error).toBeUndefined();
    });
  });

  describe('newCmdRespError', () => {
    it('creates an error response', () => {
      const resp = newCmdRespError('unauthorized', 'missing token');
      
      expect(resp.error).toEqual({
        code: 'unauthorized',
        message: 'missing token',
      });
      expect(resp.payload).toBeUndefined();
    });
  });

  describe('CmdReqBuilder', () => {
    it('builds a request with chained methods', () => {
      const req = new CmdReqBuilder('TestCmd', { data: 123 })
        .withHeader('custom', 'value')
        .withAuth('my-token')
        .build();

      expect(req.cmd).toBe('TestCmd');
      expect(req.payload).toEqual({ data: 123 });
      expect(req.headers).toEqual({
        custom: 'value',
        auth: 'my-token',
      });
    });
  });

  describe('CmdRespBuilder', () => {
    it('builds a response with chained methods', () => {
      const resp = new CmdRespBuilder({ result: 'ok' })
        .withHeader('rate-limit', '100')
        .build();

      expect(resp.payload).toEqual({ result: 'ok' });
      expect(resp.headers).toEqual({ 'rate-limit': '100' });
    });

    it('builds an error response', () => {
      const resp = new CmdRespBuilder()
        .withError('not_found', 'user not found')
        .build();

      expect(resp.error).toEqual({
        code: 'not_found',
        message: 'user not found',
      });
    });
  });

  describe('hasError', () => {
    it('returns true for error response', () => {
      const resp = newCmdRespError('error', 'message');
      expect(hasError(resp)).toBe(true);
    });

    it('returns false for success response', () => {
      const resp = newCmdResp({ data: 'ok' });
      expect(hasError(resp)).toBe(false);
    });
  });

  describe('getHeader', () => {
    it('returns header value', () => {
      const resp = newCmdResp();
      resp.headers['test'] = 'value';
      
      expect(getHeader(resp, 'test')).toBe('value');
    });

    it('returns undefined for missing header', () => {
      const resp = newCmdResp();
      expect(getHeader(resp, 'missing')).toBeUndefined();
    });
  });

  describe('context functions', () => {
    it('creates and modifies context', () => {
      let ctx = newContext();
      expect(ctx.values).toEqual({});

      const req = newCmdReq('TestCmd');
      ctx = contextWithRequest(ctx, req);
      expect(ctx.request).toBe(req);

      const resp = newCmdResp();
      ctx = contextWithResponse(ctx, resp);
      expect(ctx.response).toBe(resp);

      const user = { id: 1, name: 'Test' };
      ctx = contextWithUser(ctx, user);
      expect(ctx.user).toBe(user);
    });

    it('gets request header from context', () => {
      const req = newCmdReq('TestCmd');
      req.headers['auth'] = 'token123';
      const ctx = contextWithRequest(newContext(), req);

      expect(getRequestHeader(ctx, 'auth')).toBe('token123');
      expect(getRequestHeader(ctx, 'missing')).toBeUndefined();
    });

    it('sets response header in context', () => {
      let ctx = newContext();
      ctx = setResponseHeader(ctx, 'rate-limit', '100');

      expect(ctx.response?.headers['rate-limit']).toBe('100');
    });
  });

  describe('HEADER_PREFIX', () => {
    it('has correct value', () => {
      expect(HEADER_PREFIX).toBe('x-mvp-');
    });
  });
});
