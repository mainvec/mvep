/**
 * MVP Client for JavaScript/TypeScript
 */

import type { CmdReq, CmdResp, MvpContext } from './envelope';
import { newCmdReq, newContext, hasError } from './envelope';
import type { ClientInterceptor, ClientInvoker } from './middleware';
import { HttpTransporter, type FetchOptions, type EnvelopeTransporter, type Encoder, jsonEncoder } from './http-transport';
import type { Package } from './package';

/** Default encoder content type */
export const DEFAULT_ENCODER = 'application/json';

/**
 * ClientConfig holds configuration for the MVP client
 */
export interface ClientConfig {
  /** Base URL of the MVP server (e.g., "http://127.0.0.1:8080") */
  baseUrl: string;
  /** Base URL path for endpoints (e.g., "/api") */
  basePath?: string;
  /** Content type for encoding commands (default: "application/json") */
  encoder?: string;
  /** Timeout in milliseconds (default: 30000) */
  timeout?: number;
  /** Custom fetch implementation */
  fetch?: typeof fetch;
  /** Additional headers for all requests */
  headers?: Record<string, string>;
  /** Request credentials mode */
  credentials?: RequestCredentials;
  /** Request mode */
  mode?: RequestMode;
  /** Client interceptor chain for all outgoing requests */
  interceptor?: ClientInterceptor;
}

/**
 * PackageHandler manages command transport for a package
 */
export interface PackageHandler {
  pkg: Package;
  transporter: EnvelopeTransporter;
}

/**
 * Client represents an MVP client for communicating with an MVP server
 */
export class Client {
  private config: ClientConfig;
  private packages: Map<string, PackageClient>;
  private interceptor?: ClientInterceptor;

  constructor(config: ClientConfig) {
    if (!config.baseUrl) {
      throw new Error('base URL is required');
    }

    this.config = {
      ...config,
      basePath: config.basePath || '',
      encoder: config.encoder || DEFAULT_ENCODER,
      timeout: config.timeout || 30000,
    };
    this.packages = new Map();
    this.interceptor = config.interceptor;
  }

  /**
   * Registers an MVP package with the client
   */
  registerPackage(pkg: Package): PackageClient {
    if (!pkg) {
      throw new Error('package is required');
    }

    const pkgName = pkg.getName();
    if (this.packages.has(pkgName)) {
      throw new Error(`package already registered: ${pkgName}`);
    }

    // Build the package path: basePath + "/" + packageName + "/cmd"
    const pkgPath = `${this.config.basePath}/${pkgName}/cmd`;

    const fetchOptions: FetchOptions = {
      fetch: this.config.fetch,
      headers: this.config.headers,
      credentials: this.config.credentials,
      mode: this.config.mode,
    };

    const transporter = new HttpTransporter(
      this.config.baseUrl,
      pkgPath,
      fetchOptions
    );

    const handler: PackageHandler = {
      pkg,
      transporter,
    };

    const pkgClient = new PackageClient(this, pkg, handler, this.config.encoder!);
    this.packages.set(pkgName, pkgClient);

    return pkgClient;
  }

  /**
   * Returns a registered package client by name
   */
  getPackage(name: string): PackageClient | undefined {
    return this.packages.get(name);
  }

  /**
   * Sends a ping/health check request to the server
   */
  async ping(): Promise<string> {
    const healthPath = `${this.config.basePath}/health`;
    const url = this.config.baseUrl + healthPath;

    const fetchFn = this.config.fetch || globalThis.fetch;
    const response = await fetchFn(url, {
      method: 'GET',
      headers: this.config.headers,
      credentials: this.config.credentials,
      mode: this.config.mode,
    });

    return response.text();
  }

  /**
   * Sends a command to the appropriate package based on command type
   * The command must be registered with one of the registered packages
   */
  async sendCmd<T = unknown>(cmd: unknown): Promise<T> {
    if (!cmd) {
      throw new Error('command is required');
    }

    // Find the package that knows about this command
    for (const pkgClient of this.packages.values()) {
      const cmdName = pkgClient.pkg.nameOf(cmd);
      if (cmdName) {
        return pkgClient.sendCmd<T>(cmd);
      }
    }

    throw new Error('no registered package found for command');
  }

  /**
   * Gets the client interceptor
   */
  getInterceptor(): ClientInterceptor | undefined {
    return this.interceptor;
  }
}

/**
 * PackageClient represents a client for a specific MVP package
 */
export class PackageClient {
  private client: Client;
  public pkg: Package;
  private handler: PackageHandler;
  private encoder: string;
  private encoderImpl: Encoder;

  constructor(client: Client, pkg: Package, handler: PackageHandler, encoder: string) {
    this.client = client;
    this.pkg = pkg;
    this.handler = handler;
    this.encoder = encoder;
    this.encoderImpl = jsonEncoder; // Default to JSON encoder
  }

  /**
   * Sends a command through this package client
   */
  async sendCmd<T = unknown>(cmd: unknown): Promise<T> {
    const result = await this.sendCmdInternal(cmd, {}, this.encoder);
    return result.data as T;
  }

  /**
   * Sends a command with a specific encoder
   */
  async sendCmdWithEncoder<T = unknown>(cmd: unknown, encoder: string): Promise<T> {
    const result = await this.sendCmdInternal(cmd, {}, encoder);
    return result.data as T;
  }

  /**
   * Sends a command with headers and returns the result along with response
   */
  async sendCmdReq<T = unknown>(
    cmd: unknown,
    headers?: Record<string, string>
  ): Promise<{ data: T; response: CmdResp }> {
    return this.sendCmdInternal<T>(cmd, headers || {}, this.encoder);
  }

  /**
   * Sends a command with headers using a specific encoder
   */
  async sendCmdReqWithEncoder<T = unknown>(
    cmd: unknown,
    headers: Record<string, string>,
    encoder: string
  ): Promise<{ data: T; response: CmdResp }> {
    return this.sendCmdInternal<T>(cmd, headers, encoder);
  }

  /**
   * Internal method to send a command with optional interceptor
   */
  private async sendCmdInternal<T = unknown>(
    cmd: unknown,
    headers: Record<string, string>,
    encoder: string
  ): Promise<{ data: T; response: CmdResp }> {
    if (!cmd) {
      throw new Error('missing command');
    }

    const cmdName = this.pkg.nameOf(cmd);
    if (!cmdName) {
      throw new Error('invalid command: command not registered with package');
    }

    // Build CmdReq
    const req = newCmdReq(cmdName, cmd);
    if (headers) {
      Object.assign(req.headers, headers);
    }

    // Create context
    const ctx = newContext();

    // Define the invoker that calls the actual transport
    const invoker: ClientInvoker = async (ctx: MvpContext, req: CmdReq): Promise<CmdResp> => {
      return this.handler.transporter.transportCmdReq(req, encoder);
    };

    // Get interceptor from client
    const interceptor = this.client.getInterceptor();

    // Execute with or without interceptor
    let resp: CmdResp;
    if (interceptor) {
      resp = await interceptor(ctx, req, invoker);
    } else {
      resp = await invoker(ctx, req);
    }

    // Check for errors
    if (hasError(resp)) {
      throw new Error(`command error: [${resp.error!.code}] ${resp.error!.message}`);
    }

    // Decode response - for JSON, the payload is already decoded
    const resultName = cmdName + 'Result';
    const resultInstance = this.pkg.instanceOf(resultName);
    
    // If payload is already an object, use it directly
    let data: T;
    if (typeof resp.payload === 'object') {
      data = resp.payload as T;
    } else if (typeof resp.payload === 'string') {
      data = JSON.parse(resp.payload) as T;
    } else {
      data = resp.payload as T;
    }

    return { data, response: resp };
  }

  /**
   * Sends a raw command with bytes directly without encoding/decoding
   */
  async sendRawCmd(cmdName: string, cmdData: string): Promise<string> {
    const req = newCmdReq(cmdName, cmdData);
    const resp = await this.handler.transporter.transportCmdReq(req, this.encoder);
    
    if (hasError(resp)) {
      throw new Error(`command error: [${resp.error!.code}] ${resp.error!.message}`);
    }

    return typeof resp.payload === 'string' ? resp.payload : JSON.stringify(resp.payload);
  }

  /**
   * Sets the default encoder for this package client
   */
  setEncoder(encoder: string): void {
    this.encoder = encoder;
  }

  /**
   * Returns the current encoder for this package client
   */
  getEncoder(): string {
    return this.encoder;
  }

  /**
   * Returns the underlying MVP package
   */
  getPackage(): Package {
    return this.pkg;
  }
}

/**
 * Creates a new MVP client
 */
export function newClient(config: ClientConfig): Client {
  return new Client(config);
}
