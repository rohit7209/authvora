package repository

import (
	"context"

	"github.com/authvora/auth-service/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SigningKeyRepository handles JWT signing key persistence.
type SigningKeyRepository struct {
	pool *pgxpool.Pool
}

// NewSigningKeyRepository creates a new SigningKeyRepository.
func NewSigningKeyRepository(pool *pgxpool.Pool) *SigningKeyRepository {
	return &SigningKeyRepository{pool: pool}
}

// GetActiveSigningKey retrieves the active signing key for a tenant.
func (r *SigningKeyRepository) GetActiveSigningKey(ctx context.Context, tenantID string) (*model.SigningKey, error) {
	query := `
		SELECT id, tenant_id, kid, public_key, private_key, is_active, created_at
		FROM signing_keys WHERE tenant_id = $1 AND is_active = true
		ORDER BY created_at DESC LIMIT 1
	`
	var k model.SigningKey
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(
		&k.ID, &k.TenantID, &k.KeyID, &k.PublicKey, &k.PrivateKey, &k.Active, &k.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// GetAllActiveKeys retrieves all active signing keys for a tenant (for JWKS).
func (r *SigningKeyRepository) GetAllActiveKeys(ctx context.Context, tenantID string) ([]model.SigningKey, error) {
	query := `
		SELECT id, tenant_id, kid, public_key, private_key, is_active, created_at
		FROM signing_keys WHERE tenant_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []model.SigningKey
	for rows.Next() {
		var k model.SigningKey
		if err := rows.Scan(&k.ID, &k.TenantID, &k.KeyID, &k.PublicKey, &k.PrivateKey, &k.Active, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// CreateSigningKey creates a new signing key.
func (r *SigningKeyRepository) CreateSigningKey(ctx context.Context, key *model.SigningKey) error {
	query := `
		INSERT INTO signing_keys (id, tenant_id, kid, public_key, private_key, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		key.ID, key.TenantID, key.KeyID, key.PublicKey, key.PrivateKey, key.Active, key.CreatedAt,
	)
	return err
}
