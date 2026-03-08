package middleware

import "context"

// contextKey is an unexported type for context keys to avoid collisions.
type contextKey int

const (
	_ contextKey = iota
	contextKeyTenantID
	contextKeyUserID
	contextKeyEmail
)

// TenantIDFromContext returns the tenant ID from the request context.
func TenantIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyTenantID).(string); ok {
		return v
	}
	return ""
}

// UserIDFromContext returns the user ID from the request context.
func UserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyUserID).(string); ok {
		return v
	}
	return ""
}

// EmailFromContext returns the email from the request context.
func EmailFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyEmail).(string); ok {
		return v
	}
	return ""
}
