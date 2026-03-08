package model

import "time"

type Tenant struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserCredential struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	TenantID     string    `json:"tenant_id"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type OAuthConnection struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	UserID      string    `json:"user_id"`
	Provider    string    `json:"provider"`
	ProviderUID string    `json:"provider_uid"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Session matches the sessions table: id, user_id, tenant_id, ip_address, user_agent, device_id, expires_at, revoked, created_at
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}

// RefreshToken matches: id, token_hash, user_id, tenant_id, session_id, family_id, used, revoked, expires_at, created_at
type RefreshToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	SessionID string    `json:"session_id"`
	FamilyID  string    `json:"family_id"`
	TokenHash string    `json:"-"`
	Used      bool      `json:"used"`
	Revoked   bool      `json:"revoked"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type SigningKey struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	KeyID      string    `json:"key_id"`
	PublicKey  string    `json:"public_key"`
	PrivateKey string    `json:"-"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
}

// LoginEvent matches: id, tenant_id, user_id, event_type, ip_address, user_agent, device_id, location, metadata, risk_score, created_at
type LoginEvent struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    *string   `json:"user_id,omitempty"`
	EventType string    `json:"event_type"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Metadata  string    `json:"metadata"` // JSONB stored as string
	CreatedAt time.Time `json:"created_at"`
}

type MFAConfig struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	Secret    string    `json:"-"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SuspiciousIP struct {
	IPAddress string `json:"ip_address"`
	Count     int64  `json:"count"`
}

type Agent struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Name       string    `json:"name"`
	APIKeyHash string    `json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
