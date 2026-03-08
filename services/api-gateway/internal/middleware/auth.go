package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// JWTAuthMiddleware extracts Bearer token from Authorization header,
// parses JWT claims (sub, tid, email), and puts them into request context.
// Does not verify signature—backend services handle that.
func JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			writeAuthError(w, "UNAUTHORIZED", "Authorization header with Bearer token is required", http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		if tokenStr == "" {
			writeAuthError(w, "UNAUTHORIZED", "Bearer token is required", http.StatusUnauthorized)
			return
		}

		token, _, err := jwt.NewParser().ParseUnverified(tokenStr, jwt.MapClaims{})
		if err != nil {
			writeAuthError(w, "INVALID_TOKEN", "Invalid or malformed token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeAuthError(w, "INVALID_TOKEN", "Invalid token claims", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		if sub, ok := claims["sub"].(string); ok {
			ctx = context.WithValue(ctx, contextKeyUserID, sub)
		}
		if tid, ok := claims["tid"].(string); ok {
			ctx = context.WithValue(ctx, contextKeyTenantID, tid)
		}
		if email, ok := claims["email"].(string); ok {
			ctx = context.WithValue(ctx, contextKeyEmail, email)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeAuthError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Status  int    `json:"status"`
		}{
			Code:    code,
			Message: message,
			Status:  status,
		},
	})
}
