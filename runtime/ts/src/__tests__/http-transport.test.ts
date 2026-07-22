import { describe, it, expect, vi, beforeEach } from 'vitest';
import { HttpTransporter, HEADER_PREFIX } from '..';

describe('HttpTransporter', () => {
  const mockFetch = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('constructor', () => {
    it('throws on missing url', () => {
      expect(() => new HttpTransporter('', '/pkg', { fetch: mockFetch }))
        .toThrow('missing url');
    });

    it('throws on invalid url', () => {
      expect(() => new HttpTransporter('ftp://example.com', '/pkg', { fetch: mockFetch }))
        .toThrow('invalid url');
    });

    it('throws on missing package path', () => {
      expect(() => new HttpTransporter('http://example.com', '', { fetch: mockFetch }))
        .toThrow('missing package path');
    });

    it('creates transporter with valid config', () => {
      const transporter = new HttpTransporter('http://example.com', '/api/pkg', { fetch: mockFetch });
      expect(transporter.getUrl()).toBe('http://example.com/api/pkg');
    });
  });

  describe('transportCmd', () => {
    it('throws on missing command name', async () => {
      const transporter = new HttpTransporter('http://example.com', '/pkg', { fetch: mockFetch });
      
      await expect(transporter.transportCmd('', 'application/json', {}))
        .rejects.toThrow('missing command name');
    });

    it('sends POST request with correct headers', async () => {
      const mockResponse = {
        ok: true,
        text: vi.fn().mockResolvedValue('{"result":"ok"}'),
        headers: new Map(),
      };
      mockFetch.mockResolvedValue(mockResponse);

      const transporter = new HttpTransporter('http://example.com', '/pkg', { fetch: mockFetch });
      const result = await transporter.transportCmd('TestCmd', 'application/json', { data: 'test' });

      expect(mockFetch).toHaveBeenCalledWith(
        'http://example.com/pkg',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
            'x-mainvec-cmd': 'TestCmd',
          }),
          body: '{"data":"test"}',
        })
      );
      expect(result).toEqual({ result: 'ok' });
    });

    it('throws on non-OK response', async () => {
      const mockResponse = {
        ok: false,
        status: 500,
        headers: {
          get: vi.fn().mockReturnValue(null),
        },
      };
      mockFetch.mockResolvedValue(mockResponse);

      const transporter = new HttpTransporter('http://example.com', '/pkg', { fetch: mockFetch });

      await expect(transporter.transportCmd('TestCmd', 'application/json', {}))
        .rejects.toThrow('unexpected status code: 500');
    });

    it('throws with error message from header', async () => {
      const mockResponse = {
        ok: false,
        status: 400,
        headers: {
          get: vi.fn().mockReturnValue('validation failed'),
        },
      };
      mockFetch.mockResolvedValue(mockResponse);

      const transporter = new HttpTransporter('http://example.com', '/pkg', { fetch: mockFetch });

      await expect(transporter.transportCmd('TestCmd', 'application/json', {}))
        .rejects.toThrow('error: validation failed');
    });
  });

  describe('transportCmdReq', () => {
    it('sends request with custom headers', async () => {
      const mockHeaders = new Map<string, string>();
      mockHeaders.set('x-mvp-response-header', 'value');
      
      const mockResponse = {
        ok: true,
        text: vi.fn().mockResolvedValue('{"result":"ok"}'),
        headers: {
          forEach: (callback: (value: string, key: string) => void) => {
            mockHeaders.forEach((value, key) => callback(value, key));
          },
          get: vi.fn().mockReturnValue(null),
        },
      };
      mockFetch.mockResolvedValue(mockResponse);

      const transporter = new HttpTransporter('http://example.com', '/pkg', { fetch: mockFetch });
      const resp = await transporter.transportCmdReq(
        { cmd: 'TestCmd', headers: { 'auth': 'token123' }, payload: { data: 'test' } },
        'application/json'
      );

      expect(mockFetch).toHaveBeenCalledWith(
        'http://example.com/pkg',
        expect.objectContaining({
          headers: expect.objectContaining({
            'x-mvp-auth': 'token123',
          }),
        })
      );

      expect(resp.payload).toEqual({ result: 'ok' });
      expect(resp.headers['response-header']).toBe('value');
    });

    it('includes error info on non-OK response', async () => {
      const mockHeaders = new Map<string, string>();
      
      const mockResponse = {
        ok: false,
        status: 401,
        text: vi.fn().mockResolvedValue('unauthorized'),
        headers: {
          forEach: (callback: (value: string, key: string) => void) => {
            mockHeaders.forEach((value, key) => callback(value, key));
          },
          get: vi.fn().mockReturnValue('auth required'),
        },
      };
      mockFetch.mockResolvedValue(mockResponse);

      const transporter = new HttpTransporter('http://example.com', '/pkg', { fetch: mockFetch });
      const resp = await transporter.transportCmdReq(
        { cmd: 'TestCmd', headers: {}, payload: {} },
        'application/json'
      );

      expect(resp.error).toEqual({
        code: 'http_401',
        message: 'auth required',
      });
    });
  });
});
