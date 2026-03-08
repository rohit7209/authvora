import type { AuthvoraConfig, AuthvoraError, TokenPair } from "./types.js";

export class AuthvoraClient {
  private readonly _baseUrl: string;
  private readonly _tenantId: string;
  private readonly apiKey?: string;
  private accessToken?: string;
  private refreshToken?: string;
  private tokenExpiresAt?: number;

  readonly baseUrl: string;
  readonly tenantId: string;

  constructor(config: AuthvoraConfig) {
    this._baseUrl = config.baseUrl.replace(/\/$/, "");
    this._tenantId = config.tenantId;
    this.apiKey = config.apiKey;
    this.baseUrl = this._baseUrl;
    this.tenantId = this._tenantId;
  }

  setTokens(accessToken: string, refreshToken: string, expiresIn: number): void {
    this.accessToken = accessToken;
    this.refreshToken = refreshToken;
    this.tokenExpiresAt = Math.floor(Date.now() / 1000) + expiresIn;
  }

  clearTokens(): void {
    this.accessToken = undefined;
    this.refreshToken = undefined;
    this.tokenExpiresAt = undefined;
  }

  async getAccessToken(): Promise<string | null> {
    const now = Math.floor(Date.now() / 1000);
    const bufferSeconds = 60;

    if (this.accessToken && this.tokenExpiresAt && this.tokenExpiresAt > now + bufferSeconds) {
      return this.accessToken;
    }

    if (this.refreshToken) {
      try {
        const pair = await this.refreshTokens();
        return pair.access_token;
      } catch {
        this.clearTokens();
        return null;
      }
    }

    return this.accessToken ?? null;
  }

  private async refreshTokens(): Promise<TokenPair> {
    if (!this.refreshToken) {
      throw new Error("No refresh token available");
    }

    const response = await fetch(`${this._baseUrl}/auth/refresh`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Tenant-ID": this._tenantId,
      },
      body: JSON.stringify({ refresh_token: this.refreshToken }),
    });

    if (!response.ok) {
      const error = await this.parseErrorResponse(response);
      throw error;
    }

    const data = (await this.parseJsonResponse<TokenPair>(response)) as TokenPair;
    this.setTokens(data.access_token, data.refresh_token, data.expires_in);
    return data;
  }

  async request<T>(
    method: string,
    path: string,
    options?: {
      body?: unknown;
      headers?: Record<string, string>;
      authenticated?: boolean;
      query?: Record<string, string>;
    }
  ): Promise<T> {
    const { body, headers = {}, authenticated = false, query } = options ?? {};

    let url = `${this._baseUrl}${path.startsWith("/") ? path : `/${path}`}`;
    if (query && Object.keys(query).length > 0) {
      const searchParams = new URLSearchParams(query);
      url += `?${searchParams.toString()}`;
    }

    const requestHeaders: Record<string, string> = {
      "Content-Type": "application/json",
      "X-Tenant-ID": this._tenantId,
      ...headers,
    };

    if (this.apiKey) {
      requestHeaders["X-API-Key"] = this.apiKey;
    }

    if (authenticated) {
      const token = await this.getAccessToken();
      if (!token) {
        throw this.createError("TOKEN_REQUIRED", "Authentication required", 401);
      }
      requestHeaders["Authorization"] = `Bearer ${token}`;
    }

    const init: RequestInit = {
      method,
      headers: requestHeaders,
    };

    if (body !== undefined && body !== null) {
      init.body = JSON.stringify(body);
    }

    let response: Response;
    try {
      response = await fetch(url, init);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Network request failed";
      throw this.createError("NETWORK_ERROR", message, 0);
    }

    if (!response.ok) {
      throw await this.parseErrorResponse(response);
    }

    if (response.status === 204 || response.headers.get("content-length") === "0") {
      return undefined as T;
    }

    return this.parseJsonResponse<T>(response);
  }

  private async parseErrorResponse(response: Response): Promise<AuthvoraError> {
    let body: unknown;
    try {
      const text = await response.text();
      body = text ? JSON.parse(text) : {};
    } catch {
      body = {};
    }

    const obj = body as Record<string, unknown>;
    const error = obj.error as Record<string, unknown> | undefined;
    const code = (error?.code as string) ?? "UNKNOWN_ERROR";
    const message = (error?.message as string) ?? response.statusText ?? "Request failed";
    const status = (error?.status as number) ?? response.status;

    return this.createError(code, message, status);
  }

  private createError(code: string, message: string, status: number): AuthvoraError {
    return { code, message, status };
  }

  private async parseJsonResponse<T>(response: Response): Promise<T> {
    const text = await response.text();
    if (!text) {
      return {} as T;
    }
    try {
      return JSON.parse(text) as T;
    } catch {
      throw this.createError("INVALID_RESPONSE", "Invalid JSON response", response.status);
    }
  }
}
