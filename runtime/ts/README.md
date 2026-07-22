# @mainvec/mvp

MVEP (Mainvec Protocol) client library for JavaScript/TypeScript.

This library provides a TypeScript/JavaScript client for communicating with MVEP servers, mirroring the Go implementation in `github.com/mainvec/mvep/runtime/go/mvep`.

## Installation

```bash
npm install @mainvec/mvp
```

## Quick Start

```typescript
import { 
  newClient, 
  SimplePackage, 
  chainClient, 
  clientLoggingInterceptor,
  staticAuthHeaderInterceptor,
  createCommand
} from '@mainvec/mvp';

// Create a package with command names
const userPackage = new SimplePackage('users', ['GetUser', 'CreateUser', 'UpdateUser']);

// Create client with interceptors
const client = newClient({
  baseUrl: 'http://localhost:8080',
  basePath: '/api',
  interceptor: chainClient(
    clientLoggingInterceptor(),
    staticAuthHeaderInterceptor('my-auth-token')
  ),
});

// Register package
const users = client.registerPackage(userPackage);

// Send commands using createCommand helper
const result = await users.sendCmd(
  createCommand('GetUser', { userId: 123 })
);

console.log(result);
```

## Features

### Envelope Types

The library uses `CmdReq` and `CmdResp` envelope types for request/response handling with headers:

```typescript
import { newCmdReq, CmdReqBuilder, hasError } from '@mainvec/mvp';

// Create a request
const req = newCmdReq('GetUser', { userId: 123 });
req.headers['auth'] = 'my-token';

// Or use the builder pattern
const req2 = new CmdReqBuilder('GetUser', { userId: 123 })
  .withAuth('my-token')
  .withHeader('request-id', 'abc-123')
  .build();
```

### Interceptors

Client interceptors allow you to add cross-cutting concerns like logging, authentication, and retry logic:

```typescript
import {
  chainClient,
  clientLoggingInterceptor,
  authHeaderInterceptor,
  retryInterceptor,
  clientRequestIdInterceptor,
  generateRequestId,
} from '@mainvec/mvp';

// Chain multiple interceptors
const interceptor = chainClient(
  clientLoggingInterceptor(),                    // Log all requests/responses
  clientRequestIdInterceptor(generateRequestId), // Add request IDs
  authHeaderInterceptor(async () => 'my-token'), // Add auth headers
  retryInterceptor(3, 1000),                     // Retry failed requests
);

const client = newClient({
  baseUrl: 'http://localhost:8080',
  interceptor,
});
```

### Available Client Interceptors

- `clientLoggingInterceptor(logger?)` - Logs requests and responses with timing
- `authHeaderInterceptor(tokenProvider)` - Adds auth header using async token provider
- `staticAuthHeaderInterceptor(token)` - Adds a static auth header
- `headerInterceptor(headers)` - Adds custom headers to all requests
- `retryInterceptor(maxRetries, delayMs)` - Retries failed requests
- `clientRequestIdInterceptor(generator)` - Adds request IDs
- `timeoutInterceptor(timeoutMs)` - Adds request timeout

### Package Definition

Define your command packages using `SimplePackage` or `BasePackage`:

```typescript
// Simple package with command names
const myPackage = new SimplePackage('myservice', ['Cmd1', 'Cmd2', 'Cmd3']);

// Or use BasePackage for typed commands
class GetUserCmd {
  userId!: number;
}

class GetUserResult {
  user!: { id: number; name: string };
}

const typedPackage = new BasePackage('users')
  .registerCommandClass('GetUser', GetUserCmd, GetUserResult);
```

### Sending Commands

```typescript
// Simple command
const result = await pkgClient.sendCmd(createCommand('GetUser', { userId: 123 }));

// Command with custom headers
const { data, response } = await pkgClient.sendCmdReq(
  createCommand('GetUser', { userId: 123 }),
  { 'custom-header': 'value' }
);

// Access response headers
console.log(response.headers['rate-limit-remaining']);
```

### Health Check

```typescript
const health = await client.ping();
console.log('Server health:', health);
```

## API Reference

### Client

- `newClient(config: ClientConfig): Client` - Creates a new MVEP client
- `client.registerPackage(pkg: Package): PackageClient` - Registers a package
- `client.getPackage(name: string): PackageClient | undefined` - Gets a registered package
- `client.ping(): Promise<string>` - Health check
- `client.sendCmd(cmd: unknown): Promise<unknown>` - Sends command to appropriate package

### PackageClient

- `sendCmd<T>(cmd: unknown): Promise<T>` - Sends a command
- `sendCmdReq<T>(cmd, headers?): Promise<{data: T, response: CmdResp}>` - Sends with headers
- `sendRawCmd(cmdName, data): Promise<string>` - Sends raw data
- `setEncoder(encoder: string)` - Sets content type
- `getEncoder(): string` - Gets current content type

### Middleware

- `chainClient(...interceptors): ClientInterceptor` - Chains interceptors
- `skipCommandsClient(interceptor, ...cmds): ClientInterceptor` - Skip certain commands
- `onlyCommandsClient(interceptor, ...cmds): ClientInterceptor` - Only apply to certain commands

## Server-side Interceptors

For server-side usage (if implementing an MVEP server in Node.js):

```typescript
import {
  chain,
  loggingInterceptor,
  authInterceptor,
  recoveryInterceptor,
  requestIdInterceptor,
} from '@mainvec/mvp';

const serverInterceptor = chain(
  recoveryInterceptor(),
  loggingInterceptor(),
  requestIdInterceptor(generateRequestId),
  authInterceptor(myTokenValidator),
);
```

## Compatibility

This library is compatible with:
- Modern browsers (ES2020+)
- Node.js 18+
- Deno
- Bun

For Node.js versions without native fetch, provide a fetch implementation:

```typescript
import fetch from 'node-fetch';

const client = newClient({
  baseUrl: 'http://localhost:8080',
  fetch: fetch as unknown as typeof globalThis.fetch,
});
```

## License

MIT
