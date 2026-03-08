package repository

import (
	"context"

	"github.com/authvora/auth-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OAuthConnectionRepository struct {
	pool *pgxpool.Pool
}

func NewOAuthConnectionRepository(pool *pgxpool.Pool) *OAuthConnectionRepository {
	return &OAuthConnectionRepository{pool: pool}
}

func (r *OAuthConnectionRepository) CreateOAuthConnection(ctx context.Context, conn *model.OAuthConnection) error {
	id := uuid.New().String()
	query := `
		INSERT INTO oauth_connections (id, tenant_id, user_id, provider, provider_uid, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		id, conn.TenantID, conn.UserID, conn.Provider, conn.ProviderUID,
		conn.CreatedAt, conn.UpdatedAt,
	)
	return err
}

func (r *OAuthConnectionRepository) GetOAuthConnection(ctx context.Context, tenantID, provider, providerUID string) (*model.OAuthConnection, error) {
	query := `
		SELECT id, tenant_id, user_id, provider, provider_uid, created_at, updated_at
		FROM oauth_connections WHERE tenant_id = $1 AND provider = $2 AND provider_uid = $3
	`
	var c model.OAuthConnection
	err := r.pool.QueryRow(ctx, query, tenantID, provider, providerUID).Scan(
		&c.ID, &c.TenantID, &c.UserID, &c.Provider, &c.ProviderUID,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
