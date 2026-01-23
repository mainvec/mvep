/**
 * MVP (Mainvec Protocol) Client Library for JavaScript/TypeScript
 * 
 * This library provides a client for communicating with MVP servers,
 * mirroring the Go implementation in github.com/mainvec/mvpgo.
 * 
 * @example
 * ```typescript
 * import { newClient, SimplePackage, chainClient, clientLoggingInterceptor } from '@mainvec/mvp';
 * 
 * // Create a package with command names
 * const myPackage = new SimplePackage('myservice', ['GetUser', 'CreateUser']);
 * 
 * // Create client with interceptors
 * const client = newClient({
 *   baseUrl: 'http://localhost:8080',
 *   basePath: '/api',
 *   interceptor: chainClient(
 *     clientLoggingInterceptor(),
 *     staticAuthHeaderInterceptor('my-token')
 *   ),
 * });
 * 
 * // Register package
 * const pkgClient = client.registerPackage(myPackage);
 * 
 * // Send commands
 * const result = await pkgClient.sendCmd({ _cmdName: 'GetUser', userId: 123 });
 * ```
 */

// Envelope types
export {
  HEADER_PREFIX,
  type ErrorInfo,
  type CmdReq,
  type CmdResp,
  type MvpContext,
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
} from './envelope';

// Middleware types and utilities
export {
  type CmdHandler,
  type CmdInterceptor,
  type ClientInvoker,
  type ClientInterceptor,
  chain,
  chainClient,
  skipCommands,
  onlyCommands,
  skipCommandsClient,
  onlyCommandsClient,
} from './middleware';

// Interceptors
export {
  type Logger,
  type TokenValidator,
  type TokenProvider,
  consoleLogger,
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
} from './interceptors';

// Package types
export {
  type Package,
  type CommandDefinition,
  BasePackage,
  SimplePackage,
  createCommand,
} from './package';

// Transport
export {
  type Transporter,
  type EnvelopeTransporter,
  type Encoder,
  type FetchOptions,
  jsonEncoder,
  HttpTransporter,
  newHttpTransporter,
} from './http-transport';

// Client
export {
  DEFAULT_ENCODER,
  type ClientConfig,
  type PackageHandler,
  Client,
  PackageClient,
  newClient,
} from './client';
