package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	ipLimitPerMinute     = 100
	tenantLimitPerMinute = 1000
	rateLimitWindowSecs  = 90 // Slightly more than 1 min for cleanup
)

// NewRateLimiter returns a middleware that enforces rate limits using Redis.
// Per-IP: 100 req/min, Per-tenant: 1000 req/min.
// Gracefully degrades (allows requests) if Redis is unavailable.
func NewRateLimiter(redisClient *redis.Client) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get client IP (first in X-Forwarded-For chain if present)
			clientIP := r.Header.Get("X-Forwarded-For")
			if clientIP != "" {
				if idx := strings.Index(clientIP, ","); idx > 0 {
					clientIP = strings.TrimSpace(clientIP[:idx])
				} else {
					clientIP = strings.TrimSpace(clientIP)
				}
			}
			if clientIP == "" {
				clientIP = r.RemoteAddr
			}

			// Check IP rate limit
			if redisClient != nil {
				minute := time.Now().Unix() / 60
				ipKey := "ratelimit:ip:" + clientIP + ":" + strconv.FormatInt(minute, 10)
				allowed, err := checkLimit(ctx, redisClient, ipKey, ipLimitPerMinute)
				if err != nil {
					// Redis error: gracefully degrade, allow request
				} else if !allowed {
					writeRateLimitError(w)
					return
				}

				// Check tenant rate limit (from header; tenant middleware runs after)
				tenantID := r.Header.Get("X-Tenant-ID")
				if tenantID != "" {
					tenantKey := "ratelimit:tenant:" + tenantID + ":" + strconv.FormatInt(minute, 10)
					allowed, err := checkLimit(ctx, redisClient, tenantKey, tenantLimitPerMinute)
					if err != nil {
						// Redis error: gracefully degrade
					} else if !allowed {
						writeRateLimitError(w)
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func checkLimit(ctx context.Context, client *redis.Client, key string, limit int) (bool, error) {
	pipe := client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, rateLimitWindowSecs*time.Second)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return true, err // allow on error (graceful degradation)
	}
	count, _ := incr.Result()
	return int(count) <= limit, nil
}

func writeRateLimitError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Status  int    `json:"status"`
		} `json:"error"`
	}{
		Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Status  int    `json:"status"`
		}{
			Code:    "RATE_LIMITED",
			Message: "Too many requests",
			Status:  429,
		},
	})
}
