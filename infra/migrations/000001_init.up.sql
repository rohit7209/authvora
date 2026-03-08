-- Authvora Database Schema
-- Migration: 000001_init

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Tenants
CREATE TABLE tenants (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    slug            VARCHAR(63) NOT NULL UNIQUE,
    domain          VARCHAR(255),
    settings        JSONB NOT NULL DEFAULT '{}',
    oauth_google    JSONB,
    oauth_github    JSONB,
    oauth_apple     JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_slug ON tenants(slug);

-- Users
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email           VARCHAR(255) NOT NULL,
    email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    name            VARCHAR(255),
    avatar_url      TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_users_tenant_email UNIQUE (tenant_id, email)
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_email ON users(tenant_id, email);

-- User Credentials
CREATE TABLE user_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    credential_type VARCHAR(20) NOT NULL DEFAULT 'password',
    password_hash   TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_credentials UNIQUE (user_id, credential_type)
);

CREATE INDEX idx_user_credentials_user ON user_credentials(user_id);

-- OAuth Connections
CREATE TABLE oauth_connections (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider        VARCHAR(20) NOT NULL,
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

-- Devices (must be before sessions due to FK)
CREATE TABLE devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    fingerprint     VARCHAR(64) NOT NULL,
    device_name     VARCHAR(255),
    device_type     VARCHAR(50),
    os              VARCHAR(100),
    browser         VARCHAR(100),
    trusted         BOOLEAN NOT NULL DEFAULT FALSE,
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_devices_user_fingerprint UNIQUE (user_id, fingerprint)
);

CREATE INDEX idx_devices_user ON devices(user_id);
CREATE INDEX idx_devices_fingerprint ON devices(fingerprint);

-- Sessions
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

-- Refresh Tokens
CREATE TABLE refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash      VARCHAR(64) NOT NULL UNIQUE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    session_id      UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    family_id       UUID NOT NULL,
    used            BOOLEAN NOT NULL DEFAULT FALSE,
    revoked         BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_family ON refresh_tokens(family_id);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);

-- Signing Keys
CREATE TABLE signing_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kid             VARCHAR(64) NOT NULL UNIQUE,
    algorithm       VARCHAR(10) NOT NULL DEFAULT 'RS256',
    public_key      TEXT NOT NULL,
    private_key     TEXT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_signing_keys_tenant ON signing_keys(tenant_id);
CREATE INDEX idx_signing_keys_active ON signing_keys(tenant_id, is_active) WHERE is_active = TRUE;

-- MFA Configurations
CREATE TABLE mfa_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    method          VARCHAR(20) NOT NULL DEFAULT 'totp',
    secret          TEXT NOT NULL,
    verified        BOOLEAN NOT NULL DEFAULT FALSE,
    recovery_codes  TEXT[],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_mfa_user_method UNIQUE (user_id, method)
);

CREATE INDEX idx_mfa_user ON mfa_configs(user_id);

-- Agents
CREATE TABLE agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    agent_name      VARCHAR(255) NOT NULL,
    agent_type      VARCHAR(50) NOT NULL DEFAULT 'service',
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

-- Login Events
CREATE TABLE login_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type      VARCHAR(50) NOT NULL,
    ip_address      INET,
    user_agent      TEXT,
    device_id       UUID,
    location        JSONB,
    metadata        JSONB NOT NULL DEFAULT '{}',
    risk_score      INTEGER,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_login_events_tenant ON login_events(tenant_id);
CREATE INDEX idx_login_events_user ON login_events(user_id);
CREATE INDEX idx_login_events_type ON login_events(event_type);
CREATE INDEX idx_login_events_created ON login_events(created_at);
CREATE INDEX idx_login_events_ip ON login_events(ip_address);

-- Risk Assessments
CREATE TABLE risk_assessments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login_event_id  UUID NOT NULL REFERENCES login_events(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    risk_score      INTEGER NOT NULL CHECK (risk_score >= 0 AND risk_score <= 100),
    risk_level      VARCHAR(20) NOT NULL,
    signals         JSONB NOT NULL DEFAULT '{}',
    action_taken    VARCHAR(20) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_risk_assessments_event ON risk_assessments(login_event_id);
CREATE INDEX idx_risk_assessments_tenant ON risk_assessments(tenant_id);
CREATE INDEX idx_risk_assessments_score ON risk_assessments(risk_score);

-- IP History
CREATE TABLE ip_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    ip_address      INET NOT NULL,
    location        JSONB,
    first_seen_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    login_count     INTEGER NOT NULL DEFAULT 1,

    CONSTRAINT uq_ip_history_user_ip UNIQUE (user_id, ip_address)
);

CREATE INDEX idx_ip_history_user ON ip_history(user_id);
CREATE INDEX idx_ip_history_ip ON ip_history(ip_address);

-- Tenant Policies
CREATE TABLE tenant_policies (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE UNIQUE,
    password_min_length     INTEGER NOT NULL DEFAULT 8,
    password_require_upper  BOOLEAN NOT NULL DEFAULT FALSE,
    password_require_number BOOLEAN NOT NULL DEFAULT FALSE,
    password_require_special BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_required            BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_risk_threshold      INTEGER NOT NULL DEFAULT 50,
    session_max_age_hours   INTEGER NOT NULL DEFAULT 168,
    max_sessions_per_user   INTEGER NOT NULL DEFAULT 10,
    rate_limit_per_minute   INTEGER NOT NULL DEFAULT 100,
    login_max_attempts      INTEGER NOT NULL DEFAULT 5,
    lockout_duration_minutes INTEGER NOT NULL DEFAULT 15,
    allowed_origins         TEXT[] NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_policies_tenant ON tenant_policies(tenant_id);

-- Seed a default tenant for development
INSERT INTO tenants (id, name, slug, domain) VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Default Tenant', 'default', 'localhost');

INSERT INTO tenant_policies (tenant_id) VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11');
