package middleware

import (
	"context"
	"encoding/json"
	"net/http"
)

// errorResponse is the JSON structure for error responses.
type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Status  int    `json:"status"`
	} `json:"error"`
}

// TenantMiddleware extracts X-Tenant-ID from the request header.
// If missing, returns 400 JSON error. Otherwise stores tenant_id in context.
func TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(errorResponse{
				Error: struct {
					Code    string `json:"code"`
					Message string `json:"message"`
					Status  int    `json:"status"`
				}{
					Code:    "MISSING_TENANT",
					Message: "X-Tenant-ID header is required",
					Status:  400,
				},
			})
			return
		}
		ctx := context.WithValue(r.Context(), contextKeyTenantID, tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
