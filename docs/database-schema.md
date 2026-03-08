# Authvora Database Schema

## Overview

PostgreSQL is the primary data store. All tables use UUID primary keys and include `tenant_id` for multi-tenant isolation. Timestamps use `TIMESTAMPTZ`.

Redis is used for session caching and rate limiting.

## Entity Relationship

```
tenants
  │
  ├── users
  │     ├── user_credentials
  │     ├── oauth_connections
  │     ├── sessions
  │     ├── refresh_tokens
  │     ├── mfa_configs
  │     ├── devices
  │     └── ip_history
  │
  ├── signing_keys
  ├── agents
  ├── login_events
  │     └── risk_assessments
  └── tenant_policies
```

## Table Definitions

### tenants

```sql
CREATE TABLE tenants (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    slug            VARCHAR(63) NOT NULL UNIQUE,
    domain          VARCHAR(255),
    settings        JSONB NOT NULL DEFAULT '{}',
    oauth_google    JSONB,  -- {client_id, client_secret, redirect_uri}
    oauth_github    JSONB,
    oauth_apple     JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
```

### users

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email           VARCHAR(255) NOT NULL,
    email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    name            VARCHAR(255),
    avatar_url      TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',  -- active, suspended, deleted
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_users_tenant_email UNIQUE (tenant_id, email)
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_email ON users(tenant_id, email);
```

### user_credentials

Separated from users to support multiple auth methods per user.

```sql
CREATE TABLE user_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    credential_type VARCHAR(20) NOT NULL DEFAULT 'password',  -- password, passkey
    password_hash   TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_credentials UNIQUE (user_id, credential_type)
);

CREATE INDEX idx_user_credentials_user ON user_credentials(user_id);
```

### oauth_connections

```sql
CREATE TABLE oauth_connections (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider        VARCHAR(20) NOT NULL,  -- google, github, apple
    provider_uid    VARCHAR(255) NOT NULL,
    access_token    TEXT,
    refresh_token   TEXT,
    token_expires   TIMESTAMPTZ,
    profile         JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_oauth_tenant_provider_uid UNIQUE (tenant_id, provider, provider_uid)
);

CREATE INDEX idx_oauth_user ON oauth_connections(user_id);
CREATE INDEX idx_oauth_provider ON oauth_connections(tenant_id, provider, provider_uid);
```

### sessions

```sql
CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    ip_address      INET,
    user_agent      TEXT,
    device_id       UUID REFERENCES devices(id),
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked         BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_tenant ON sessions(tenant_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at) WHERE revoked = FALSE;
```

### refresh_tokens

Family-based tracking for replay detection.

```sql
CREATE TABLE refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash      VARCHAR(64) NOT NULL UNIQUE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    session_id      UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    family_id       UUID NOT NULL,  -- groups related tokens for replay detection
    used            BOOLEAN NOT NULL DEFAULT FALSE,
    revoked         BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_family ON refresh_tokens(family_id);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
```

### signing_keys

Per-tenant RSA key pairs for JWT signing.

```sql
CREATE TABLE signing_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kid             VARCHAR(64) NOT NULL UNIQUE,  -- key ID for JWKS
    algorithm       VARCHAR(10) NOT NULL DEFAULT 'RS256',
    public_key      TEXT NOT NULL,
    private_key     TEXT NOT NULL,  -- encrypted at rest
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_signing_keys_tenant ON signing_keys(tenant_id);
CREATE INDEX idx_signing_keys_active ON signing_keys(tenant_id, is_active) WHERE is_active = TRUE;
```

### mfa_configs

```sql
CREATE TABLE mfa_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    method          VARCHAR(20) NOT NULL DEFAULT 'totp',  -- totp, sms
    secret          TEXT NOT NULL,  -- encrypted TOTP secret
    verified        BOOLEAN NOT NULL DEFAULT FALSE,
    recovery_codes  TEXT[],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_mfa_user_method UNIQUE (user_id, method)
);

CREATE INDEX idx_mfa_user ON mfa_configs(user_id);
```

### agents

AI agent identities.

```sql
CREATE TABLE agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    agent_name      VARCHAR(255) NOT NULL,
    agent_type      VARCHAR(50) NOT NULL DEFAULT 'service',  -- service, assistant, monitor
    api_key_hash    VARCHAR(64) NOT NULL,
    scopes          TEXT[] NOT NULL DEFAULT '{}',
    public_key      TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_agents_tenant_name UNIQUE (tenant_id, agent_name)
);

CREATE INDEX idx_agents_tenant ON agents(tenant_id);
CREATE INDEX idx_agents_api_key ON agents(api_key_hash);
```

### login_events

Append-only audit log. Serves as the event bus for MVP.

```sql
CREATE TABLE login_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type      VARCHAR(50) NOT NULL,  -- login_success, login_failure, register, oauth_login, token_refresh, logout, mfa_challenge, mfa_verify
    ip_address      INET,
    user_agent      TEXT,
    device_id       UUID,
    location        JSONB,  -- {country, region, city, lat, lon}
    metadata        JSONB NOT NULL DEFAULT '{}',
    risk_score      INTEGER,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_login_events_tenant ON login_events(tenant_id);
CREATE INDEX idx_login_events_user ON login_events(user_id);
CREATE INDEX idx_login_events_type ON login_events(event_type);
CREATE INDEX idx_login_events_created ON login_events(created_at);
CREATE INDEX idx_login_events_ip ON login_events(ip_address);
```

### risk_assessments

```sql
CREATE TABLE risk_assessments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login_event_id  UUID NOT NULL REFERENCES login_events(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    risk_score      INTEGER NOT NULL CHECK (risk_score >= 0 AND risk_score <= 100),
    risk_level      VARCHAR(20) NOT NULL,  -- low, medium, high, critical
    signals         JSONB NOT NULL DEFAULT '{}',  -- {ip_risk, geo_risk, time_risk, device_risk, travel_risk}
    action_taken    VARCHAR(20) NOT NULL,  -- allow, mfa_required, block
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_risk_assessments_event ON risk_assessments(login_event_id);
CREATE INDEX idx_risk_assessments_tenant ON risk_assessments(tenant_id);
CREATE INDEX idx_risk_assessments_score ON risk_assessments(risk_score);
```

### devices

```sql
CREATE TABLE devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    fingerprint     VARCHAR(64) NOT NULL,
    device_name     VARCHAR(255),
    device_type     VARCHAR(50),  -- desktop, mobile, tablet
    os              VARCHAR(100),
    browser         VARCHAR(100),
    trusted         BOOLEAN NOT NULL DEFAULT FALSE,
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_devices_user_fingerprint UNIQUE (user_id, fingerprint)
);

CREATE INDEX idx_devices_user ON devices(user_id);
CREATE INDEX idx_devices_fingerprint ON devices(fingerprint);
```

### ip_history

```sql
CREATE TABLE ip_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    ip_address      INET NOT NULL,
    location        JSONB,  -- {country, region, city, lat, lon}
    first_seen_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    login_count     INTEGER NOT NULL DEFAULT 1,

    CONSTRAINT uq_ip_history_user_ip UNIQUE (user_id, ip_address)
);

CREATE INDEX idx_ip_history_user ON ip_history(user_id);
CREATE INDEX idx_ip_history_ip ON ip_history(ip_address);
```

### tenant_policies

```sql
CREATE TABLE tenant_policies (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE UNIQUE,
    password_min_length     INTEGER NOT NULL DEFAULT 8,
    password_require_upper  BOOLEAN NOT NULL DEFAULT FALSE,
    password_require_number BOOLEAN NOT NULL DEFAULT FALSE,
    password_require_special BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_required            BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_risk_threshold      INTEGER NOT NULL DEFAULT 50,
    session_max_age_hours   INTEGER NOT NULL DEFAULT 168,  -- 7 days
    max_sessions_per_user   INTEGER NOT NULL DEFAULT 10,
    rate_limit_per_minute   INTEGER NOT NULL DEFAULT 100,
    login_max_attempts      INTEGER NOT NULL DEFAULT 5,
    lockout_duration_minutes INTEGER NOT NULL DEFAULT 15,
    allowed_origins         TEXT[] NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_policies_tenant ON tenant_policies(tenant_id);
```

## Redis Key Patterns

```
session:{session_id}              -> JSON session data (TTL: session duration)
rate:ip:{ip_address}              -> counter (TTL: 1 minute)
rate:tenant:{tenant_id}           -> counter (TTL: 1 minute)
rate:login:{tenant_id}:{email}    -> counter (TTL: 15 minutes)
jwks:{tenant_id}                  -> cached JWKS response (TTL: 5 minutes)
```
