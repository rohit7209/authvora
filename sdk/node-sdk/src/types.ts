export interface AuthvoraConfig {
  baseUrl: string; // API gateway URL (e.g., "https://auth.example.com")
  tenantId: string; // Tenant UUID
  apiKey?: string; // Optional API key for server-side operations
}

export interface User {
  id: string;
  email: string;
  name: string | null;
  email_verified: boolean;
  avatar_url: string | null;
  created_at: string;
  updated_at: string;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface MFARequiredResponse {
  mfa_required: true;
  mfa_token: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface RiskScore {
  risk_score: number;
  risk_level: "low" | "medium" | "high" | "critical";
  action: "allow" | "mfa_required" | "block";
  signals: {
    ip_risk: number;
    geo_risk: number;
    time_risk: number;
    device_risk: number;
    travel_risk: number;
  };
}

export interface LoginMetrics {
  period: string;
  total_logins: number;
  successful_logins: number;
  failed_logins: number;
  unique_users: number;
  mfa_challenges: number;
  blocked_attempts: number;
}

export interface AuthvoraError {
  code: string;
  message: string;
  status: number;
}

export interface RegisterParams {
  email: string;
  password: string;
  name?: string;
}

export interface LoginParams {
  email: string;
  password: string;
}

export interface OAuthGoogleParams {
  code: string;
  redirect_uri: string;
}

// For Express/Connect middleware
export interface AuthvoraRequest {
  authvora?: {
    user: User;
    token: string;
    tenantId: string;
  };
}
