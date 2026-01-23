/**
 * HTTP Transport for MVP client
 */

import type { CmdReq, CmdResp, ErrorInfo } from './envelope';
import { HEADER_PREFIX, newCmdResp } from './envelope';

/**
 * Transporter interface for sending commands
 */
export interface Transporter {
  /**
   * Transports a command and returns the response
   */
  transportCmd(
    cmdName: string,
    contentType: string,
    payload: unknown
  ): Promise<unknown>;
}

/**
 * EnvelopeTransporter interface for sending commands with headers
 */
export interface EnvelopeTransporter {
  /**
   * Transports a command request with headers and returns a response with headers
   */
  transportCmdReq(req: CmdReq, contentType: string): Promise<CmdResp>;
}

/**
 * Encoder interface for serializing/deserializing data
 */
export interface Encoder {
  /** MIME type for the encoding */
  mimeType: string;
  /** Encodes data to string or bytes */
  encode(data: unknown): string | Uint8Array;
  /** Decodes string or bytes to data */
  decode<T>(data: string | Uint8Array): T;
}

/**
 * JSON encoder implementation
 */
export const jsonEncoder: Encoder = {
  mimeType: 'application/json',
  encode: (data: unknown) => JSON.stringify(data),
  decode: <T>(data: string | Uint8Array): T => {
    const str = typeof data === 'string' ? data : new TextDecoder().decode(data);
    return JSON.parse(str) as T;
  },
};

/**
 * FetchOptions for customizing fetch behavior
 */
export interface FetchOptions {
  /** Custom fetch implementation (useful for Node.js or testing) */
  fetch?: typeof fetch;
  /** Additional headers to include in all requests */
  headers?: Record<string, string>;
  /** Request credentials mode */
  credentials?: RequestCredentials;
  /** Request mode */
  mode?: RequestMode;
}

/**
 * HttpTransporter implements HTTP-based command transport
 */
export class HttpTransporter implements Transporter, EnvelopeTransporter {
  private url: string;
  private path: string;
  private fetchFn: typeof fetch;
  private defaultHeaders: Record<string, string>;
  private credentials?: RequestCredentials;
  private mode?: RequestMode;

  constructor(
    url: string,
    pkgPath: string,
    options: FetchOptions = {}
  ) {
    if (!url) {
      throw new Error('missing url');
    }
    if (!url.startsWith('http://') && !url.startsWith('https://')) {
      throw new Error('invalid url: must start with http:// or https://');
    }
    if (!pkgPath) {
      throw new Error('missing package path');
    }

    this.url = url;
    this.path = url + pkgPath;
    this.fetchFn = options.fetch || globalThis.fetch;
    this.defaultHeaders = options.headers || {};
    this.credentials = options.credentials;
    this.mode = options.mode;

    if (!this.fetchFn) {
      throw new Error('fetch is not available. Please provide a fetch implementation.');
    }
  }

  /**
   * Gets the full URL for requests
   */
  getUrl(): string {
    return this.path;
  }

  /**
   * Transports a raw command without envelope
   */
  async transportCmd(
    cmdName: string,
    contentType: string,
    payload: unknown
  ): Promise<unknown> {
    if (!cmdName) {
      throw new Error('missing command name');
    }
    if (!contentType) {
      throw new Error('missing content type');
    }

    const body = typeof payload === 'string' ? payload : JSON.stringify(payload);

    const response = await this.fetchFn(this.path, {
      method: 'POST',
      headers: {
        ...this.defaultHeaders,
        'Content-Type': contentType,
        'x-mainvec-cmd': cmdName,
      },
      body,
      credentials: this.credentials,
      mode: this.mode,
    });

    if (!response.ok) {
      const errorMsg = response.headers.get('x-mainvec-error');
      if (errorMsg) {
        throw new Error(`error: ${errorMsg}`);
      }
      throw new Error(`unexpected status code: ${response.status}`);
    }

    const responseText = await response.text();
    try {
      return JSON.parse(responseText);
    } catch {
      return responseText;
    }
  }

  /**
   * Transports a command request with headers and returns a response with headers
   */
  async transportCmdReq(req: CmdReq, contentType: string): Promise<CmdResp> {
    if (!req) {
      throw new Error('missing command request');
    }
    if (!req.cmd) {
      throw new Error('missing command name');
    }
    if (!contentType) {
      throw new Error('missing content type');
    }

    // Build headers
    const headers: Record<string, string> = {
      ...this.defaultHeaders,
      'Content-Type': contentType,
      'x-mainvec-cmd': req.cmd,
    };

    // Add custom headers with prefix
    for (const [key, value] of Object.entries(req.headers)) {
      headers[HEADER_PREFIX + key] = value;
    }

    // Encode payload
    const body = typeof req.payload === 'string' 
      ? req.payload 
      : JSON.stringify(req.payload);

    const response = await this.fetchFn(this.path, {
      method: 'POST',
      headers,
      body,
      credentials: this.credentials,
      mode: this.mode,
    });

    // Read response body
    const responseText = await response.text();
    let payload: unknown;
    try {
      payload = JSON.parse(responseText);
    } catch {
      payload = responseText;
    }

    // Build CmdResp
    const cmdResp = newCmdResp(payload);

    // Extract response headers with prefix
    response.headers.forEach((value, key) => {
      const lowerKey = key.toLowerCase();
      if (lowerKey.startsWith(HEADER_PREFIX)) {
        const headerKey = lowerKey.slice(HEADER_PREFIX.length);
        cmdResp.headers[headerKey] = value;
      }
    });

    // Check for errors
    if (!response.ok) {
      const errorMsg = response.headers.get('x-mainvec-error') || responseText;
      const error: ErrorInfo = {
        code: `http_${response.status}`,
        message: errorMsg,
      };
      cmdResp.error = error;
    }

    return cmdResp;
  }
}

/**
 * Creates a new HttpTransporter
 */
export function newHttpTransporter(
  url: string,
  pkgPath: string,
  options?: FetchOptions
): HttpTransporter {
  return new HttpTransporter(url, pkgPath, options);
}
