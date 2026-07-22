import { describe, it, expect, vi, beforeEach } from 'vitest';
import { 
  Client, 
  newClient, 
  SimplePackage, 
  createCommand,
  chainClient,
  staticAuthHeaderInterceptor,
} from '..';

describe('Client', () => {
  const mockFetch = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('constructor', () => {
    it('throws on missing baseUrl', () => {
      expect(() => newClient({ baseUrl: '' }))
        .toThrow('base URL is required');
    });

    it('creates client with valid config', () => {
      const client = newClient({
        baseUrl: 'http://localhost:8080',
        basePath: '/api',
        fetch: mockFetch,
      });

      expect(client).toBeInstanceOf(Client);
    });
  });

  describe('registerPackage', () => {
    it('throws on null package', () => {
      const client = newClient({
        baseUrl: 'http://localhost:8080',
        fetch: mockFetch,
      });

      expect(() => client.registerPackage(null as any))
        .toThrow('package is required');
    });

    it('throws on duplicate package', () => {
      const client = newClient({
        baseUrl: 'http://localhost:8080',
        fetch: mockFetch,
      });

      const pkg = new SimplePackage('myservice', ['Cmd1']);
      client.registerPackage(pkg);

      expect(() => client.registerPackage(pkg))
        .toThrow('package already registered: myservice');
    });

    it('registers package and returns PackageClient', () => {
      const client = newClient({
        baseUrl: 'http://localhost:8080',
        fetch: mockFetch,
      });

      const pkg = new SimplePackage('myservice', ['Cmd1']);
      const pkgClient = client.registerPackage(pkg);

      expect(pkgClient).toBeDefined();
      expect(pkgClient.getPackage()).toBe(pkg);
    });
  });

  describe('getPackage', () => {
    it('returns registered package', () => {
      const client = newClient({
        baseUrl: 'http://localhost:8080',
        fetch: mockFetch,
      });

      const pkg = new SimplePackage('myservice', ['Cmd1']);
      const registered = client.registerPackage(pkg);

      expect(client.getPackage('myservice')).toBe(registered);
    });

    it('returns undefined for unregistered package', () => {
      const client = newClient({
        baseUrl: 'http://localhost:8080',
        fetch: mockFetch,
      });

      expect(client.getPackage('unknown')).toBeUndefined();
    });
  });

  describe('ping', () => {
    it('sends health check request', async () => {
      mockFetch.mockResolvedValue({
        text: vi.fn().mockResolvedValue('OK'),
      });

      const client = newClient({
        baseUrl: 'http://localhost:8080',
        basePath: '/api',
        fetch: mockFetch,
      });

      const result = await client.ping();

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/health',
        expect.objectContaining({ method: 'GET' })
      );
      expect(result).toBe('OK');
    });
  });

  describe('sendCmd', () => {
    it('throws on null command', async () => {
      const client = newClient({
        baseUrl: 'http://localhost:8080',
        fetch: mockFetch,
      });

      await expect(client.sendCmd(null))
        .rejects.toThrow('command is required');
    });

    it('throws when no package handles command', async () => {
      const client = newClient({
        baseUrl: 'http://localhost:8080',
        fetch: mockFetch,
      });

      await expect(client.sendCmd({ _cmdName: 'Unknown' }))
        .rejects.toThrow('no registered package found for command');
    });

    it('routes command to correct package', async () => {
      const mockHeaders = new Map<string, string>();
      mockFetch.mockResolvedValue({
        ok: true,
        text: vi.fn().mockResolvedValue('{"userId":123,"name":"Test"}'),
        headers: {
          forEach: (cb: (v: string, k: string) => void) => mockHeaders.forEach((v, k) => cb(v, k)),
          get: vi.fn().mockReturnValue(null),
        },
      });

      const client = newClient({
        baseUrl: 'http://localhost:8080',
        basePath: '/api',
        fetch: mockFetch,
      });

      const pkg = new SimplePackage('users', ['GetUser']);
      client.registerPackage(pkg);

      const result = await client.sendCmd(createCommand('GetUser', { userId: 123 }));

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/users/cmd',
        expect.any(Object)
      );
      expect(result).toEqual({ userId: 123, name: 'Test' });
    });
  });

  describe('PackageClient', () => {
    describe('sendCmd', () => {
      it('sends command and returns result', async () => {
        const mockHeaders = new Map<string, string>();
        mockFetch.mockResolvedValue({
          ok: true,
          text: vi.fn().mockResolvedValue('{"success":true}'),
          headers: {
            forEach: (cb: (v: string, k: string) => void) => mockHeaders.forEach((v, k) => cb(v, k)),
            get: vi.fn().mockReturnValue(null),
          },
        });

        const client = newClient({
          baseUrl: 'http://localhost:8080',
          fetch: mockFetch,
        });

        const pkg = new SimplePackage('myservice', ['DoSomething']);
        const pkgClient = client.registerPackage(pkg);

        const result = await pkgClient.sendCmd(createCommand('DoSomething', { input: 'test' }));

        expect(result).toEqual({ success: true });
      });

      it('throws on command error', async () => {
        const mockHeaders = new Map<string, string>();
        mockFetch.mockResolvedValue({
          ok: false,
          status: 400,
          text: vi.fn().mockResolvedValue('validation failed'),
          headers: {
            forEach: (cb: (v: string, k: string) => void) => mockHeaders.forEach((v, k) => cb(v, k)),
            get: vi.fn().mockReturnValue('validation failed'),
          },
        });

        const client = newClient({
          baseUrl: 'http://localhost:8080',
          fetch: mockFetch,
        });

        const pkg = new SimplePackage('myservice', ['DoSomething']);
        const pkgClient = client.registerPackage(pkg);

        await expect(pkgClient.sendCmd(createCommand('DoSomething', {})))
          .rejects.toThrow('command error: [http_400] validation failed');
      });
    });

    describe('sendCmdReq', () => {
      it('sends command with headers and returns response', async () => {
        const mockHeaders = new Map<string, string>();
        mockHeaders.set('x-mvp-rate-limit', '100');
        
        mockFetch.mockResolvedValue({
          ok: true,
          text: vi.fn().mockResolvedValue('{"success":true}'),
          headers: {
            forEach: (cb: (v: string, k: string) => void) => mockHeaders.forEach((v, k) => cb(v, k)),
            get: vi.fn().mockReturnValue(null),
          },
        });

        const client = newClient({
          baseUrl: 'http://localhost:8080',
          fetch: mockFetch,
        });

        const pkg = new SimplePackage('myservice', ['DoSomething']);
        const pkgClient = client.registerPackage(pkg);

        const { data, response } = await pkgClient.sendCmdReq(
          createCommand('DoSomething', {}),
          { 'custom-header': 'value' }
        );

        expect(data).toEqual({ success: true });
        expect(response.headers['rate-limit']).toBe('100');
      });
    });

    describe('with interceptor', () => {
      it('applies client interceptor', async () => {
        const mockHeaders = new Map<string, string>();
        mockFetch.mockResolvedValue({
          ok: true,
          text: vi.fn().mockResolvedValue('{"success":true}'),
          headers: {
            forEach: (cb: (v: string, k: string) => void) => mockHeaders.forEach((v, k) => cb(v, k)),
            get: vi.fn().mockReturnValue(null),
          },
        });

        const client = newClient({
          baseUrl: 'http://localhost:8080',
          fetch: mockFetch,
          interceptor: staticAuthHeaderInterceptor('my-token'),
        });

        const pkg = new SimplePackage('myservice', ['DoSomething']);
        const pkgClient = client.registerPackage(pkg);

        await pkgClient.sendCmd(createCommand('DoSomething', {}));

        expect(mockFetch).toHaveBeenCalledWith(
          expect.any(String),
          expect.objectContaining({
            headers: expect.objectContaining({
              'x-mvp-auth': 'my-token',
            }),
          })
        );
      });
    });
  });
});
