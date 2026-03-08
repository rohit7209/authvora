package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/authvora/auth-service/internal/handler"
	"github.com/authvora/auth-service/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware extracts the Bearer token from Authorization header,
// decodes JWT claims (without full verification - verification done at gateway level),
// and puts user_id and tenant_id into context.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			writeAuthError(w, "missing_token", "Authorization Bearer token required")
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, _, err := jwt.NewParser().ParseUnverified(tokenStr, jwt.MapClaims{})
		if err != nil {
			writeAuthError(w, "invalid_token", "Invalid token format")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeAuthError(w, "invalid_token", "Invalid token claims")
			return
		}

		sub, _ := claims["sub"].(string)
		tid, _ := claims["tid"].(string)
		if sub == "" || tid == "" {
			writeAuthError(w, "invalid_token", "Missing sub or tid claims")
			return
		}

		ctx := handler.WithUserID(r.Context(), sub)
		ctx = handler.WithTenantID(ctx, tid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeAuthError(w http.ResponseWriter, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(model.ErrorResponse{
		Code:    code,
		Message: message,
		Status:  http.StatusUnauthorized,
	})
}
