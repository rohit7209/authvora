package repository

import (
	"context"

	"github.com/authvora/auth-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository handles user persistence.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// CreateUser creates a new user.
func (r *UserRepository) CreateUser(ctx context.Context, tenantID, email, name string) (*model.User, error) {
	id := uuid.New().String()
	query := `
		INSERT INTO users (id, tenant_id, email, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, tenant_id, email, name, created_at, updated_at
	`
	var u model.User
	err := r.pool.QueryRow(ctx, query, id, tenantID, email, name).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.Name, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByID retrieves a user by ID.
func (r *UserRepository) GetUserByID(ctx context.Context, tenantID, userID string) (*model.User, error) {
	query := `
		SELECT id, tenant_id, email, name, COALESCE(avatar_url, ''), created_at, updated_at
		FROM users WHERE id = $1 AND tenant_id = $2
	`
	var u model.User
	err := r.pool.QueryRow(ctx, query, userID, tenantID).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.Name, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByEmail retrieves a user by email.
func (r *UserRepository) GetUserByEmail(ctx context.Context, tenantID, email string) (*model.User, error) {
	query := `
		SELECT id, tenant_id, email, name, COALESCE(avatar_url, ''), created_at, updated_at
		FROM users WHERE email = $1 AND tenant_id = $2
	`
	var u model.User
	err := r.pool.QueryRow(ctx, query, email, tenantID).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.Name, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateUser updates a user.
func (r *UserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users SET name = $1, avatar_url = $2, updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4
	`
	_, err := r.pool.Exec(ctx, query, user.Name, user.AvatarURL, user.ID, user.TenantID)
	return err
}
