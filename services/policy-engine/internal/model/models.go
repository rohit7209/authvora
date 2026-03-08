package model

import "time"

// TenantPolicy represents a tenant's security policy.
type TenantPolicy struct {
	ID                      string    `json:"id"`
	TenantID                string    `json:"tenant_id"`
	PasswordMinLength       int       `json:"password_min_length"`
	PasswordRequireUpper    bool      `json:"password_require_upper"`
	PasswordRequireNumber   bool      `json:"password_require_number"`
	PasswordRequireSpecial  bool      `json:"password_require_special"`
	MFARequired             bool      `json:"mfa_required"`
	MFARiskThreshold        int       `json:"mfa_risk_threshold"`
	SessionMaxAgeHours      int       `json:"session_max_age_hours"`
	MaxSessionsPerUser      int       `json:"max_sessions_per_user"`
	RateLimitPerMinute      int       `json:"rate_limit_per_minute"`
	LoginMaxAttempts        int       `json:"login_max_attempts"`
	LockoutDurationMinutes  int       `json:"lockout_duration_minutes"`
	AllowedOrigins          []string  `json:"allowed_origins"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// UpdatePolicyRequest contains optional fields for partial policy updates.
type UpdatePolicyRequest struct {
	PasswordMinLength       *int      `json:"password_min_length,omitempty"`
	PasswordRequireUpper   *bool     `json:"password_require_upper,omitempty"`
	PasswordRequireNumber  *bool     `json:"password_require_number,omitempty"`
	PasswordRequireSpecial *bool     `json:"password_require_special,omitempty"`
	MFARequired            *bool     `json:"mfa_required,omitempty"`
	MFARiskThreshold       *int      `json:"mfa_risk_threshold,omitempty"`
	SessionMaxAgeHours     *int      `json:"session_max_age_hours,omitempty"`
	MaxSessionsPerUser     *int      `json:"max_sessions_per_user,omitempty"`
	RateLimitPerMinute     *int      `json:"rate_limit_per_minute,omitempty"`
	LoginMaxAttempts       *int      `json:"login_max_attempts,omitempty"`
	LockoutDurationMinutes *int      `json:"lockout_duration_minutes,omitempty"`
	AllowedOrigins         *[]string `json:"allowed_origins,omitempty"`
}

// PolicyResponse is the JSON response for policy endpoints.
type PolicyResponse struct {
	ID                      string    `json:"id"`
	TenantID                string    `json:"tenant_id"`
	PasswordMinLength       int       `json:"password_min_length"`
	PasswordRequireUpper    bool      `json:"password_require_upper"`
	PasswordRequireNumber   bool      `json:"password_require_number"`
	PasswordRequireSpecial  bool      `json:"password_require_special"`
	MFARequired             bool      `json:"mfa_required"`
	MFARiskThreshold        int       `json:"mfa_risk_threshold"`
	SessionMaxAgeHours      int       `json:"session_max_age_hours"`
	MaxSessionsPerUser      int       `json:"max_sessions_per_user"`
	RateLimitPerMinute      int       `json:"rate_limit_per_minute"`
	LoginMaxAttempts        int       `json:"login_max_attempts"`
	LockoutDurationMinutes  int       `json:"lockout_duration_minutes"`
	AllowedOrigins          []string  `json:"allowed_origins"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// ValidatePasswordRequest is the request body for password validation.
type ValidatePasswordRequest struct {
	Password string `json:"password"`
	TenantID string `json:"tenant_id"`
}

// ValidatePasswordResponse is the response for password validation.
type ValidatePasswordResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
}

// ErrorResponse is the JSON structure for error responses.
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Status  int    `json:"status"`
	} `json:"error"`
}
