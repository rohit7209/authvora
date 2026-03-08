# Authvora API Design

## Base URL

All endpoints are served through the API Gateway at `http://localhost:8080`.

All requests to tenant-scoped endpoints must include the `X-Tenant-ID` header.

## Authentication

Protected endpoints require an `Authorization: Bearer <access_token>` header.

## Endpoints

---

### POST /api/v1/auth/register

Register a new user with email and password.

**Headers:**
- `X-Tenant-ID: <tenant_id>` (required)

**Request:**
```json
{
  "email": "user@example.com",
  "password": "securepassword123",
  "name": "John Doe"
}
```

**Response (201):**
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "John Doe",
    "email_verified": false,
    "created_at": "2025-01-01T00:00:00Z"
  },
  "access_token": "eyJhbGciOi...",
  "refresh_token": "dGhpcyBpcyBh...",
  "expires_in": 900
}
```

**Errors:**
- `400` Invalid input
- `409` Email already registered

---

### POST /api/v1/auth/login

Authenticate with email and password.

**Headers:**
- `X-Tenant-ID: <tenant_id>` (required)

**Request:**
```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response (200):**
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "John Doe"
  },
  "access_token": "eyJhbGciOi...",
  "refresh_token": "dGhpcyBpcyBh...",
  "expires_in": 900,
  "mfa_required": false
}
```

**Response (200, MFA required):**
```json
{
  "mfa_required": true,
  "mfa_token": "temporary-mfa-token"
}
```

**Errors:**
- `401` Invalid credentials
- `403` Account suspended
- `423` Account locked (too many attempts)
- `429` Rate limited

---

### POST /api/v1/auth/oauth/google

Exchange a Google OAuth authorization code for tokens.

**Headers:**
- `X-Tenant-ID: <tenant_id>` (required)

**Request:**
```json
{
  "code": "google-auth-code",
  "redirect_uri": "https://app.example.com/callback"
}
```

**Response (200):**
```json
{
  "user": {
    "id": "uuid",
    "email": "user@gmail.com",
    "name": "John Doe",
    "avatar_url": "https://lh3.googleusercontent.com/..."
  },
  "access_token": "eyJhbGciOi...",
  "refresh_token": "dGhpcyBpcyBh...",
  "expires_in": 900,
  "is_new_user": true
}
```

---

### POST /api/v1/auth/token/refresh

Refresh an access token using a refresh token.

**Request:**
```json
{
  "refresh_token": "dGhpcyBpcyBh..."
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "bmV3IHJlZnJl...",
  "expires_in": 900
}
```

**Errors:**
- `401` Invalid or expired refresh token
- `401` Token reuse detected (entire family revoked)

---

### POST /api/v1/auth/token/revoke

Revoke a session/refresh token.

**Headers:**
- `Authorization: Bearer <access_token>`

**Request:**
```json
{
  "refresh_token": "dGhpcyBpcyBh..."
}
```

**Response (204):** No content

---

### GET /api/v1/auth/.well-known/jwks.json

Get public keys for JWT verification.

**Headers:**
- `X-Tenant-ID: <tenant_id>` (required)

**Response (200):**
```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "key-id-1",
      "use": "sig",
      "alg": "RS256",
      "n": "base64url-encoded-modulus",
      "e": "AQAB"
    }
  ]
}
```

---

### POST /api/v1/auth/mfa/setup

Set up MFA for the authenticated user.

**Headers:**
- `Authorization: Bearer <access_token>`

**Request:**
```json
{
  "method": "totp"
}
```

**Response (200):**
```json
{
  "secret": "BASE32SECRET",
  "uri": "otpauth://totp/Authvora:user@example.com?secret=BASE32SECRET&issuer=Authvora",
  "recovery_codes": ["code1", "code2", "code3", "code4", "code5"]
}
```

---

### POST /api/v1/auth/mfa/verify

Verify an MFA code during login or setup.

**Request:**
```json
{
  "mfa_token": "temporary-mfa-token",
  "code": "123456"
}
```

**Response (200):**
```json
{
  "user": { ... },
  "access_token": "eyJhbGciOi...",
  "refresh_token": "dGhpcyBpcyBh...",
  "expires_in": 900
}
```

---

### GET /api/v1/users/me

Get the authenticated user's profile.

**Headers:**
- `Authorization: Bearer <access_token>`

**Response (200):**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "name": "John Doe",
  "email_verified": true,
  "avatar_url": null,
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-01T00:00:00Z"
}
```

---

### GET /api/v1/users/:id

Get a user by ID (requires admin scope).

**Headers:**
- `Authorization: Bearer <access_token>`

**Response (200):** Same as `/users/me`

---

### GET /api/v1/risk/evaluate

Evaluate risk for a given context.

**Headers:**
- `Authorization: Bearer <access_token>`

**Query Parameters:**
- `user_id` (required)
- `ip_address` (required)
- `user_agent` (optional)
- `device_fingerprint` (optional)

**Response (200):**
```json
{
  "risk_score": 42,
  "risk_level": "medium",
  "action": "mfa_required",
  "signals": {
    "ip_risk": 20,
    "geo_risk": 60,
    "time_risk": 10,
    "device_risk": 30,
    "travel_risk": 0
  },
  "details": {
    "ip_country": "DE",
    "usual_country": "IN",
    "is_vpn": false,
    "is_new_device": true
  }
}
```

---

### GET /api/v1/analytics/login-metrics

Get login statistics for the tenant.

**Headers:**
- `Authorization: Bearer <access_token>`

**Query Parameters:**
- `period` (optional): `24h`, `7d`, `30d` (default: `24h`)

**Response (200):**
```json
{
  "period": "24h",
  "total_logins": 1523,
  "successful_logins": 1401,
  "failed_logins": 122,
  "unique_users": 834,
  "mfa_challenges": 45,
  "blocked_attempts": 12,
  "top_failure_reasons": [
    {"reason": "invalid_password", "count": 89},
    {"reason": "account_locked", "count": 18},
    {"reason": "risk_blocked", "count": 15}
  ]
}
```

---

### GET /api/v1/analytics/suspicious-ips

Get flagged IP addresses.

**Headers:**
- `Authorization: Bearer <access_token>`

**Query Parameters:**
- `limit` (optional, default: 50)

**Response (200):**
```json
{
  "suspicious_ips": [
    {
      "ip_address": "198.51.100.1",
      "failed_attempts": 142,
      "unique_accounts_targeted": 38,
      "first_seen": "2025-01-01T00:00:00Z",
      "last_seen": "2025-01-01T12:00:00Z",
      "country": "XX",
      "risk_score": 95
    }
  ]
}
```

---

### POST /api/v1/simulate/attack

Run an attack simulation for security testing.

**Headers:**
- `Authorization: Bearer <access_token>`

**Request:**
```json
{
  "attack_type": "credential_stuffing",
  "config": {
    "num_attempts": 1000,
    "target_accounts": 50,
    "source_ips": 10,
    "duration_seconds": 60
  }
}
```

**Response (200):**
```json
{
  "simulation_id": "uuid",
  "attack_type": "credential_stuffing",
  "status": "completed",
  "results": {
    "total_attempts": 1000,
    "blocked": 950,
    "rate_limited": 42,
    "flagged_suspicious": 8,
    "detection_rate": 0.95,
    "mean_detection_time_ms": 12
  }
}
```

---

## JWT Token Claims

```json
{
  "sub": "user-uuid",
  "tid": "tenant-uuid",
  "email": "user@example.com",
  "iat": 1700000000,
  "exp": 1700000900,
  "iss": "authvora",
  "aud": "tenant-slug",
  "jti": "token-uuid"
}
```

## Error Response Format

All errors follow this structure:

```json
{
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "The provided email or password is incorrect.",
    "status": 401
  }
}
```

## Error Codes

| Code | HTTP Status | Description |
|------|------------|-------------|
| INVALID_INPUT | 400 | Request validation failed |
| INVALID_CREDENTIALS | 401 | Wrong email or password |
| TOKEN_EXPIRED | 401 | Access token expired |
| TOKEN_INVALID | 401 | Token is malformed or invalid |
| REFRESH_TOKEN_REUSED | 401 | Refresh token replay detected |
| FORBIDDEN | 403 | Insufficient permissions |
| ACCOUNT_SUSPENDED | 403 | Account is suspended |
| NOT_FOUND | 404 | Resource not found |
| EMAIL_EXISTS | 409 | Email already registered |
| ACCOUNT_LOCKED | 423 | Too many failed attempts |
| RATE_LIMITED | 429 | Too many requests |
| INTERNAL_ERROR | 500 | Internal server error |
