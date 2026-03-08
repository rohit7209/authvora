package repository

import (
	"context"

	"github.com/authvora/auth-service/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (r *RefreshTokenRepository) CreateRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, token_hash, user_id, tenant_id, session_id, family_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		token.ID, token.TokenHash, token.UserID, token.TenantID,
		token.SessionID, token.FamilyID, token.ExpiresAt, token.CreatedAt,
	)
	return err
}

func (r *RefreshTokenRepository) GetRefreshTokenByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	query := `
		SELECT id, token_hash, user_id, tenant_id, session_id, family_id, used, revoked, expires_at, created_at
		FROM refresh_tokens WHERE token_hash = $1
	`
	var t model.RefreshToken
	err := r.pool.QueryRow(ctx, query, hash).Scan(
		&t.ID, &t.TokenHash, &t.UserID, &t.TenantID, &t.SessionID, &t.FamilyID,
		&t.Used, &t.Revoked, &t.ExpiresAt, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *RefreshTokenRepository) MarkRefreshTokenUsed(ctx context.Context, tokenID string) error {
	query := `UPDATE refresh_tokens SET used = true WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, tokenID)
	return err
}

func (r *RefreshTokenRepository) RevokeRefreshTokenFamily(ctx context.Context, familyID string) error {
	query := `UPDATE refresh_tokens SET revoked = true WHERE family_id = $1`
	_, err := r.pool.Exec(ctx, query, familyID)
	return err
}
