package repository

import (
	"context"

	"github.com/authvora/auth-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CredentialRepository handles user credential persistence.
type CredentialRepository struct {
	pool *pgxpool.Pool
}

// NewCredentialRepository creates a new CredentialRepository.
func NewCredentialRepository(pool *pgxpool.Pool) *CredentialRepository {
	return &CredentialRepository{pool: pool}
}

// CreateCredential creates a credential for a user.
func (r *CredentialRepository) CreateCredential(ctx context.Context, userID, tenantID, passwordHash string) error {
	id := uuid.New().String()
	query := `
		INSERT INTO user_credentials (id, user_id, tenant_id, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query, id, userID, tenantID, passwordHash)
	return err
}

// GetCredentialByUserID retrieves the credential for a user.
func (r *CredentialRepository) GetCredentialByUserID(ctx context.Context, userID string) (*model.UserCredential, error) {
	query := `
		SELECT id, user_id, tenant_id, password_hash, created_at, updated_at
		FROM user_credentials WHERE user_id = $1
	`
	var c model.UserCredential
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&c.ID, &c.UserID, &c.TenantID, &c.PasswordHash, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
