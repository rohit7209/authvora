import type {
  AuthResponse,
  MFARequiredResponse,
  RegisterParams,
  LoginParams,
  OAuthGoogleParams,
  TokenPair,
  User,
} from "./types.js";
import type { AuthvoraClient } from "./client.js";

export class AuthvoraAuth {
  constructor(private client: AuthvoraClient) {}

  async register(params: RegisterParams): Promise<AuthResponse> {
    const response = await this.client.request<AuthResponse>("POST", "/auth/register", {
      body: params,
      authenticated: false,
    });
    this.client.setTokens(response.access_token, response.refresh_token, response.expires_in);
    return response;
  }

  async login(params: LoginParams): Promise<AuthResponse | MFARequiredResponse> {
    const response = await this.client.request<AuthResponse | MFARequiredResponse>(
      "POST",
      "/auth/login",
      {
        body: params,
        authenticated: false,
      }
    );

    if ("mfa_required" in response && response.mfa_required) {
      return response;
    }

    const authResponse = response as AuthResponse;
    this.client.setTokens(
      authResponse.access_token,
      authResponse.refresh_token,
      authResponse.expires_in
    );
    return authResponse;
  }

  async loginWithGoogle(params: OAuthGoogleParams): Promise<AuthResponse> {
    const response = await this.client.request<AuthResponse>("POST", "/auth/google", {
      body: params,
      authenticated: false,
    });
    this.client.setTokens(response.access_token, response.refresh_token, response.expires_in);
    return response;
  }

  async refreshToken(refreshToken: string): Promise<TokenPair> {
    const response = await this.client.request<TokenPair>("POST", "/auth/refresh", {
      body: { refresh_token: refreshToken },
      authenticated: false,
    });
    this.client.setTokens(response.access_token, response.refresh_token, response.expires_in);
    return response;
  }

  async logout(refreshToken: string): Promise<void> {
    await this.client.request<void>("POST", "/auth/logout", {
      body: { refresh_token: refreshToken },
      authenticated: false,
    });
  }

  async verifyMFA(mfaToken: string, code: string): Promise<AuthResponse> {
    const response = await this.client.request<AuthResponse>("POST", "/auth/mfa/verify", {
      body: { mfa_token: mfaToken, code },
      authenticated: false,
    });
    this.client.setTokens(response.access_token, response.refresh_token, response.expires_in);
    return response;
  }

  async getUser(): Promise<User> {
    return this.client.request<User>("GET", "/auth/me", { authenticated: true });
  }

  async getUserById(userId: string): Promise<User> {
    return this.client.request<User>("GET", `/auth/users/${encodeURIComponent(userId)}`, {
      authenticated: true,
    });
  }

  async getJWKS(): Promise<Record<string, unknown>> {
    return this.client.request<Record<string, unknown>>("GET", "/auth/jwks", {
      authenticated: false,
    });
  }
}
