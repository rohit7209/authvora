package repository

import (
	"context"
	"fmt"

	"github.com/authvora/policy-engine/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PolicyRepository handles persistence of tenant policies.
type PolicyRepository struct {
	pool *pgxpool.Pool
}

// NewPolicyRepository creates a new PolicyRepository.
func NewPolicyRepository(pool *pgxpool.Pool) *PolicyRepository {
	return &PolicyRepository{pool: pool}
}

// GetPolicyByTenantID retrieves a policy by tenant ID.
func (r *PolicyRepository) GetPolicyByTenantID(ctx context.Context, tenantID string) (*model.TenantPolicy, error) {
	query := `
		SELECT id, tenant_id, password_min_length, password_require_upper, password_require_number,
		       password_require_special, mfa_required, mfa_risk_threshold, session_max_age_hours,
		       max_sessions_per_user, rate_limit_per_minute, login_max_attempts, lockout_duration_minutes,
		       allowed_origins, created_at, updated_at
		FROM tenant_policies
		WHERE tenant_id = $1
	`
	var p model.TenantPolicy
	var allowedOrigins []string
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(
		&p.ID, &p.TenantID, &p.PasswordMinLength, &p.PasswordRequireUpper, &p.PasswordRequireNumber,
		&p.PasswordRequireSpecial, &p.MFARequired, &p.MFARiskThreshold, &p.SessionMaxAgeHours,
		&p.MaxSessionsPerUser, &p.RateLimitPerMinute, &p.LoginMaxAttempts, &p.LockoutDurationMinutes,
		&allowedOrigins, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query policy: %w", err)
	}
	if allowedOrigins == nil {
		p.AllowedOrigins = []string{}
	} else {
		p.AllowedOrigins = allowedOrigins
	}
	return &p, nil
}

// CreatePolicy creates a new policy with defaults for the given tenant.
func (r *PolicyRepository) CreatePolicy(ctx context.Context, tenantID string) (*model.TenantPolicy, error) {
	query := `
		INSERT INTO tenant_policies (tenant_id)
		VALUES ($1)
		RETURNING id, tenant_id, password_min_length, password_require_upper, password_require_number,
		          password_require_special, mfa_required, mfa_risk_threshold, session_max_age_hours,
		          max_sessions_per_user, rate_limit_per_minute, login_max_attempts, lockout_duration_minutes,
		          allowed_origins, created_at, updated_at
	`
	var p model.TenantPolicy
	var allowedOrigins []string
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(
		&p.ID, &p.TenantID, &p.PasswordMinLength, &p.PasswordRequireUpper, &p.PasswordRequireNumber,
		&p.PasswordRequireSpecial, &p.MFARequired, &p.MFARiskThreshold, &p.SessionMaxAgeHours,
		&p.MaxSessionsPerUser, &p.RateLimitPerMinute, &p.LoginMaxAttempts, &p.LockoutDurationMinutes,
		&allowedOrigins, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert policy: %w", err)
	}
	if allowedOrigins == nil {
		p.AllowedOrigins = []string{}
	} else {
		p.AllowedOrigins = allowedOrigins
	}
	return &p, nil
}

// UpdatePolicy performs a partial update of a policy.
func (r *PolicyRepository) UpdatePolicy(ctx context.Context, tenantID string, req model.UpdatePolicyRequest) (*model.TenantPolicy, error) {
	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if req.PasswordMinLength != nil {
		updates = append(updates, fmt.Sprintf("password_min_length = $%d", argNum))
		args = append(args, *req.PasswordMinLength)
		argNum++
	}
	if req.PasswordRequireUpper != nil {
		updates = append(updates, fmt.Sprintf("password_require_upper = $%d", argNum))
		args = append(args, *req.PasswordRequireUpper)
		argNum++
	}
	if req.PasswordRequireNumber != nil {
		updates = append(updates, fmt.Sprintf("password_require_number = $%d", argNum))
		args = append(args, *req.PasswordRequireNumber)
		argNum++
	}
	if req.PasswordRequireSpecial != nil {
		updates = append(updates, fmt.Sprintf("password_require_special = $%d", argNum))
		args = append(args, *req.PasswordRequireSpecial)
		argNum++
	}
	if req.MFARequired != nil {
		updates = append(updates, fmt.Sprintf("mfa_required = $%d", argNum))
		args = append(args, *req.MFARequired)
		argNum++
	}
	if req.MFARiskThreshold != nil {
		updates = append(updates, fmt.Sprintf("mfa_risk_threshold = $%d", argNum))
		args = append(args, *req.MFARiskThreshold)
		argNum++
	}
	if req.SessionMaxAgeHours != nil {
		updates = append(updates, fmt.Sprintf("session_max_age_hours = $%d", argNum))
		args = append(args, *req.SessionMaxAgeHours)
		argNum++
	}
	if req.MaxSessionsPerUser != nil {
		updates = append(updates, fmt.Sprintf("max_sessions_per_user = $%d", argNum))
		args = append(args, *req.MaxSessionsPerUser)
		argNum++
	}
	if req.RateLimitPerMinute != nil {
		updates = append(updates, fmt.Sprintf("rate_limit_per_minute = $%d", argNum))
		args = append(args, *req.RateLimitPerMinute)
		argNum++
	}
	if req.LoginMaxAttempts != nil {
		updates = append(updates, fmt.Sprintf("login_max_attempts = $%d", argNum))
		args = append(args, *req.LoginMaxAttempts)
		argNum++
	}
	if req.LockoutDurationMinutes != nil {
		updates = append(updates, fmt.Sprintf("lockout_duration_minutes = $%d", argNum))
		args = append(args, *req.LockoutDurationMinutes)
		argNum++
	}
	if req.AllowedOrigins != nil {
		updates = append(updates, fmt.Sprintf("allowed_origins = $%d", argNum))
		args = append(args, *req.AllowedOrigins)
		argNum++
	}

	if len(updates) == 0 {
		return r.GetPolicyByTenantID(ctx, tenantID)
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, tenantID)

	query := fmt.Sprintf(`
		UPDATE tenant_policies
		SET %s
		WHERE tenant_id = $%d
		RETURNING id, tenant_id, password_min_length, password_require_upper, password_require_number,
		          password_require_special, mfa_required, mfa_risk_threshold, session_max_age_hours,
		          max_sessions_per_user, rate_limit_per_minute, login_max_attempts, lockout_duration_minutes,
		          allowed_origins, created_at, updated_at
	`, joinStrings(updates, ", "), argNum)

	var p model.TenantPolicy
	var allowedOrigins []string
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&p.ID, &p.TenantID, &p.PasswordMinLength, &p.PasswordRequireUpper, &p.PasswordRequireNumber,
		&p.PasswordRequireSpecial, &p.MFARequired, &p.MFARiskThreshold, &p.SessionMaxAgeHours,
		&p.MaxSessionsPerUser, &p.RateLimitPerMinute, &p.LoginMaxAttempts, &p.LockoutDurationMinutes,
		&allowedOrigins, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update policy: %w", err)
	}
	if allowedOrigins == nil {
		p.AllowedOrigins = []string{}
	} else {
		p.AllowedOrigins = allowedOrigins
	}
	return &p, nil
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for i := 1; i < len(ss); i++ {
		result += sep + ss[i]
	}
	return result
}
