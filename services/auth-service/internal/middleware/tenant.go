package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/authvora/auth-service/internal/handler"
	"github.com/authvora/auth-service/internal/model"
)

// TenantMiddleware extracts X-Tenant-ID from the request and puts it in context.
// Returns 400 if the header is missing.
func TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(model.ErrorResponse{
				Code:    "missing_tenant",
				Message: "X-Tenant-ID header required",
				Status:  http.StatusBadRequest,
			})
			return
		}
		ctx := handler.WithTenantID(r.Context(), tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
