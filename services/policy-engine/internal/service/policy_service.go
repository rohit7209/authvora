package service

import (
	"context"
	"errors"
	"fmt"
	"unicode"

	"github.com/authvora/policy-engine/internal/model"
	"github.com/authvora/policy-engine/internal/repository"
	"github.com/jackc/pgx/v5"
)

// PolicyService provides policy business logic.
type PolicyService struct {
	repo *repository.PolicyRepository
}

// NewPolicyService creates a new PolicyService.
func NewPolicyService(repo *repository.PolicyRepository) *PolicyService {
	return &PolicyService{repo: repo}
}

// GetPolicy returns the policy for a tenant, creating a default one if it does not exist.
func (s *PolicyService) GetPolicy(ctx context.Context, tenantID string) (*model.TenantPolicy, error) {
	policy, err := s.repo.GetPolicyByTenantID(ctx, tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			policy, createErr := s.repo.CreatePolicy(ctx, tenantID)
			if createErr != nil {
				return nil, fmt.Errorf("create policy: %w", createErr)
			}
			return policy, nil
		}
		return nil, fmt.Errorf("get policy: %w", err)
	}
	return policy, nil
}

// UpdatePolicy updates a tenant's policy with the given request.
func (s *PolicyService) UpdatePolicy(ctx context.Context, tenantID string, req model.UpdatePolicyRequest) (*model.TenantPolicy, error) {
	_, err := s.repo.GetPolicyByTenantID(ctx, tenantID)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		_, createErr := s.repo.CreatePolicy(ctx, tenantID)
		if createErr != nil {
			return nil, fmt.Errorf("ensure policy exists: %w", createErr)
		}
	} else if err != nil {
		return nil, fmt.Errorf("get policy: %w", err)
	}
	return s.repo.UpdatePolicy(ctx, tenantID, req)
}

// ValidatePassword validates a password against the tenant's policy.
// Returns all failed rules, not just the first one.
func (s *PolicyService) ValidatePassword(ctx context.Context, tenantID string, password string) (*model.ValidatePasswordResponse, error) {
	policy, err := s.GetPolicy(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var errors []string

	if len(password) < policy.PasswordMinLength {
		errors = append(errors, fmt.Sprintf("password must be at least %d characters", policy.PasswordMinLength))
	}

	if policy.PasswordRequireUpper {
		hasUpper := false
		for _, r := range password {
			if unicode.IsUpper(r) {
				hasUpper = true
				break
			}
		}
		if !hasUpper {
			errors = append(errors, "password must contain at least one uppercase letter")
		}
	}

	if policy.PasswordRequireNumber {
		hasNumber := false
		for _, r := range password {
			if unicode.IsNumber(r) {
				hasNumber = true
				break
			}
		}
		if !hasNumber {
			errors = append(errors, "password must contain at least one number")
		}
	}

	if policy.PasswordRequireSpecial {
		hasSpecial := false
		for _, r := range password {
			if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
				hasSpecial = true
				break
			}
		}
		if !hasSpecial {
			errors = append(errors, "password must contain at least one special character")
		}
	}

	return &model.ValidatePasswordResponse{
		Valid:  len(errors) == 0,
		Errors: errors,
	}, nil
}

// EvaluateMFARequirement determines whether MFA is required for the given tenant and risk score.
func (s *PolicyService) EvaluateMFARequirement(ctx context.Context, tenantID string, riskScore int) (bool, error) {
	policy, err := s.GetPolicy(ctx, tenantID)
	if err != nil {
		return false, err
	}

	if policy.MFARequired {
		return true, nil
	}
	if riskScore >= policy.MFARiskThreshold {
		return true, nil
	}
	return false, nil
}
