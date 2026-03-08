package repository

import (
	"context"

	"github.com/authvora/auth-service/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantRepository handles tenant persistence.
type TenantRepository struct {
	pool *pgxpool.Pool
}

// NewTenantRepository creates a new TenantRepository.
func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

// GetTenantByID retrieves a tenant by ID.
func (r *TenantRepository) GetTenantByID(ctx context.Context, tenantID string) (*model.Tenant, error) {
	query := `
		SELECT id, slug, name, created_at, updated_at
		FROM tenants WHERE id = $1
	`
	var t model.Tenant
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(
		&t.ID, &t.Slug, &t.Name, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
