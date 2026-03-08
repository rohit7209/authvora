package repository

import (
	"context"

	"github.com/authvora/auth-service/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

func (r *SessionRepository) CreateSession(ctx context.Context, session *model.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, tenant_id, ip_address, user_agent, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		session.ID, session.UserID, session.TenantID,
		nilIfEmptyStr(session.IPAddress), session.UserAgent, session.ExpiresAt, session.CreatedAt,
	)
	return err
}

func (r *SessionRepository) GetSessionByID(ctx context.Context, sessionID string) (*model.Session, error) {
	query := `
		SELECT id, user_id, tenant_id, COALESCE(ip_address::text, ''), COALESCE(user_agent, ''), expires_at, revoked, created_at
		FROM sessions WHERE id = $1
	`
	var s model.Session
	err := r.pool.QueryRow(ctx, query, sessionID).Scan(
		&s.ID, &s.UserID, &s.TenantID,
		&s.IPAddress, &s.UserAgent, &s.ExpiresAt, &s.Revoked, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepository) RevokeSession(ctx context.Context, sessionID string) error {
	query := `UPDATE sessions SET revoked = true WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, sessionID)
	return err
}

func nilIfEmptyStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
