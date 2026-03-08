package repository

import (
	"context"

	"github.com/authvora/auth-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LoginMetrics struct {
	TotalLogins      int64    `json:"total_logins"`
	SuccessfulLogins int64    `json:"successful_logins"`
	FailedLogins     int64    `json:"failed_logins"`
	UniqueUsers      int64    `json:"unique_users"`
	TopFailureReasons []FailureReason `json:"top_failure_reasons"`
}

type FailureReason struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

type EventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{pool: pool}
}

func (r *EventRepository) CreateLoginEvent(ctx context.Context, event *model.LoginEvent) error {
	id := uuid.New().String()
	query := `
		INSERT INTO login_events (id, tenant_id, user_id, event_type, ip_address, user_agent, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
	`
	metadata := event.Metadata
	if metadata == "" {
		metadata = "{}"
	}
	_, err := r.pool.Exec(ctx, query,
		id, event.TenantID, event.UserID, event.EventType,
		nilIfEmpty(event.IPAddress), event.UserAgent, metadata, event.CreatedAt,
	)
	return err
}

func (r *EventRepository) GetLoginMetrics(ctx context.Context, tenantID, period string) (*LoginMetrics, error) {
	interval := periodToInterval(period)
	metrics := &LoginMetrics{}

	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM login_events
		WHERE tenant_id = $1 AND created_at > NOW() - $2::interval
	`, tenantID, interval).Scan(&metrics.TotalLogins)
	if err != nil {
		return nil, err
	}

	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM login_events
		WHERE tenant_id = $1 AND event_type = 'login_success' AND created_at > NOW() - $2::interval
	`, tenantID, interval).Scan(&metrics.SuccessfulLogins)
	if err != nil {
		return nil, err
	}

	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM login_events
		WHERE tenant_id = $1 AND event_type = 'login_failure' AND created_at > NOW() - $2::interval
	`, tenantID, interval).Scan(&metrics.FailedLogins)
	if err != nil {
		return nil, err
	}

	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT user_id) FROM login_events
		WHERE tenant_id = $1 AND user_id IS NOT NULL AND created_at > NOW() - $2::interval
	`, tenantID, interval).Scan(&metrics.UniqueUsers)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT metadata->>'failure_reason' as reason, COUNT(*) as cnt
		FROM login_events
		WHERE tenant_id = $1 AND event_type = 'login_failure'
			AND metadata->>'failure_reason' IS NOT NULL
			AND created_at > NOW() - $2::interval
		GROUP BY reason ORDER BY cnt DESC LIMIT 10
	`, tenantID, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var fr FailureReason
		if err := rows.Scan(&fr.Reason, &fr.Count); err != nil {
			return nil, err
		}
		metrics.TopFailureReasons = append(metrics.TopFailureReasons, fr)
	}

	return metrics, rows.Err()
}

func (r *EventRepository) GetSuspiciousIPs(ctx context.Context, tenantID string, limit int, period string) ([]model.SuspiciousIP, error) {
	if limit <= 0 {
		limit = 10
	}
	interval := periodToInterval(period)
	query := `
		SELECT ip_address::text, COUNT(*) as cnt FROM login_events
		WHERE tenant_id = $1 AND event_type = 'login_failure'
			AND ip_address IS NOT NULL
			AND created_at > NOW() - $2::interval
		GROUP BY ip_address ORDER BY cnt DESC LIMIT $3
	`
	rows, err := r.pool.Query(ctx, query, tenantID, interval, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ips []model.SuspiciousIP
	for rows.Next() {
		var s model.SuspiciousIP
		if err := rows.Scan(&s.IPAddress, &s.Count); err != nil {
			return nil, err
		}
		ips = append(ips, s)
	}
	return ips, rows.Err()
}

func periodToInterval(period string) string {
	switch period {
	case "7d":
		return "7 days"
	case "30d":
		return "30 days"
	default:
		return "24 hours"
	}
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
