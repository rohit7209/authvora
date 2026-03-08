# Authvora System Architecture

## Overview

Authvora is an AI-native authentication infrastructure platform. It provides authentication, identity intelligence, fraud detection, security observability, and AI-agent access via MCP.

## System Topology

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLIENT LAYER                             │
│  ┌──────────────────────┐     ┌──────────────────────────────┐  │
│  │  Web / Mobile / APIs │     │  AI Agents (Claude, Cursor)  │  │
│  └──────────┬───────────┘     └──────────────┬───────────────┘  │
│             │                                │                  │
│  ┌──────────▼───────────┐     ┌──────────────▼───────────────┐  │
│  │      Node SDK        │     │         MCP Server           │  │
│  └──────────┬───────────┘     └──────────────┬───────────────┘  │
└─────────────┼────────────────────────────────┼──────────────────┘
              │                                │
              ▼                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                       API GATEWAY (:8080)                       │
│  Rate Limiting │ Tenant Resolution │ JWT Validation │ Logging   │
└───────┬────────────────┬──────────────────────┬─────────────────┘
        │                │                      │
        ▼                ▼                      ▼
┌──────────────┐ ┌──────────────┐  ┌────────────────────┐
│ Auth Service │ │Policy Engine │  │    Risk Engine      │
│   (:8081)    │ │  (:8083)     │  │     (:8082)         │
│              │ │              │  │                      │
│ Registration │ │ Tenant Rules │  │ Risk Scoring         │
│ Login        │ │ MFA Policies │  │ Anomaly Detection    │
│ OAuth        │ │ Authz Rules  │  │ Impossible Travel    │
│ JWT/JWKS     │ │              │  │ IP Reputation        │
│ Sessions     │ │              │  │                      │
│ MFA          │ │              │  │                      │
└──────┬───────┘ └──────┬───────┘  └──────────┬───────────┘
       │                │                     │
       ▼                ▼                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                         DATA LAYER                              │
│  ┌──────────────┐  ┌─────────────┐  ┌────────────────────────┐ │
│  │  PostgreSQL   │  │    Redis    │  │   Event Log (PG)       │ │
│  │  (Identity)   │  │  (Sessions) │  │   (Audit Trail)        │ │
│  └──────────────┘  └─────────────┘  └────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Service Boundaries

### Auth Service (Go, :8081)

The core authentication service. Handles all identity operations:

- **User Registration**: Email/password with argon2id hashing
- **Authentication**: Email/password login, Google OAuth
- **JWT Infrastructure**: RS256 token issuance, JWKS endpoint, per-tenant signing keys
- **Session Management**: Redis-backed sessions, refresh token rotation with family tracking
- **MFA**: TOTP-based multi-factor authentication
- **User Directory**: CRUD operations on user records

All operations are tenant-scoped. Every database query filters by `tenant_id`.

### API Gateway (Go, :8080)

The single entry point for all API traffic:

- **Reverse Proxy**: Routes requests to backend services based on path prefix
- **Tenant Resolution**: Extracts tenant from `X-Tenant-ID` header
- **JWT Validation**: Validates access tokens on protected routes
- **Rate Limiting**: Token-bucket rate limiter per IP and per tenant, backed by Redis
- **Request Logging**: Structured logging of all requests

Routing rules:
- `/api/v1/auth/*` -> Auth Service (:8081)
- `/api/v1/users/*` -> Auth Service (:8081)
- `/api/v1/risk/*` -> Risk Engine (:8082)
- `/api/v1/policies/*` -> Policy Engine (:8083)
- `/api/v1/analytics/*` -> Auth Service (:8081)
- `/api/v1/simulate/*` -> Risk Engine (:8082)

### Risk Engine (Python/FastAPI, :8082)

AI-powered security analysis:

- **Risk Scoring**: Evaluates login attempts on a 0-100 scale
- **Signal Analysis**: IP reputation, geo-location, device fingerprint, login time
- **Impossible Travel**: Detects physically impossible location changes between logins
- **Action Thresholds**: 0-30 allow, 31-70 require MFA, 71-100 block

### Policy Engine (Go, :8083)

Tenant-level configuration and authorization:

- **Authentication Policies**: Password complexity, session duration, MFA enforcement
- **Rate Limit Policies**: Per-tenant rate limit configuration
- **Authorization Rules**: Role-based access control

### MCP Server (TypeScript, :8084)

AI agent integration layer:

- **Tool Exposure**: Exposes Authvora capabilities as MCP tools
- **Agent Authentication**: API key and JWT-based agent identity verification
- **Scope Enforcement**: Restricts tool access based on agent permissions
- **Audit Trail**: Logs all agent actions for compliance

## Authentication Flows

### Email/Password Registration

```
Client -> POST /api/v1/auth/register {email, password, tenant_id}
  -> Gateway: resolve tenant, rate limit
  -> Auth Service: validate input
  -> Auth Service: hash password (argon2id)
  -> Auth Service: create user record
  -> Auth Service: create credential record
  -> Auth Service: issue JWT + refresh token
  -> Auth Service: log login event
  <- Response: {access_token, refresh_token, user}
```

### Email/Password Login

```
Client -> POST /api/v1/auth/login {email, password, tenant_id}
  -> Gateway: resolve tenant, rate limit
  -> Auth Service: lookup user by email + tenant
  -> Auth Service: verify password hash
  -> Risk Engine: evaluate login risk
  -> If risk > 70: block login
  -> If risk > 30 && MFA enabled: require MFA
  -> Auth Service: create session in Redis
  -> Auth Service: issue JWT + refresh token
  -> Auth Service: log login event
  <- Response: {access_token, refresh_token, user}
```

### Google OAuth

```
Client -> POST /api/v1/auth/oauth/google {code, tenant_id}
  -> Gateway: resolve tenant
  -> Auth Service: exchange code for Google tokens
  -> Auth Service: fetch Google user profile
  -> Auth Service: find or create user
  -> Auth Service: link OAuth connection
  -> Auth Service: issue JWT + refresh token
  <- Response: {access_token, refresh_token, user}
```

### Token Refresh

```
Client -> POST /api/v1/auth/token/refresh {refresh_token}
  -> Auth Service: validate refresh token
  -> Auth Service: check token family (replay detection)
  -> Auth Service: rotate refresh token (invalidate old, issue new)
  -> Auth Service: issue new access token
  <- Response: {access_token, refresh_token}
```

## Security Architecture

### Token Security

- **Access tokens**: RS256 JWT, 15-minute expiry, per-tenant signing keys
- **Refresh tokens**: Opaque tokens stored in database, 7-day expiry
- **Token family**: Each refresh token chain tracked; if a used token is replayed, the entire family is revoked
- **JWKS**: Public keys served at `/.well-known/jwks.json` for distributed verification

### Password Security

- **Algorithm**: argon2id (memory: 64MB, iterations: 3, parallelism: 2)
- **Validation**: Minimum 8 characters, configurable per-tenant policy

### Rate Limiting

- **Per-IP**: 100 requests/minute default
- **Per-tenant**: 1000 requests/minute default
- **Login-specific**: 5 failed attempts per account per 15 minutes

### Key Rotation

- Signing keys rotated on schedule (default: 90 days)
- Old keys remain in JWKS for token verification until all issued tokens expire
- New key marked as primary for signing

## Multi-Tenancy Model

Row-level isolation using `tenant_id` on every table:

- Every API request must include tenant identification
- All database queries filter by `tenant_id`
- Composite unique constraints include `tenant_id` (e.g., email uniqueness is per-tenant)
- Each tenant has independent signing keys, OAuth config, and policies

## Data Flow

### Login Event Pipeline

```
Login Attempt
  -> Auth Service processes login
  -> Insert into login_events table (append-only)
  -> Risk Engine reads events for scoring
  -> Observability APIs aggregate for dashboards
```

For MVP, the event pipeline uses PostgreSQL as an append-only log. Kafka integration planned for v2 to support real-time streaming and external consumers.
