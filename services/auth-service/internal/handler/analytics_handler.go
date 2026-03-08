package handler

import (
	"net/http"
	"strconv"

	"github.com/authvora/auth-service/internal/repository"
)

// AnalyticsHandler handles analytics HTTP requests.
type AnalyticsHandler struct {
	eventRepo *repository.EventRepository
}

// NewAnalyticsHandler creates a new AnalyticsHandler.
func NewAnalyticsHandler(eventRepo *repository.EventRepository) *AnalyticsHandler {
	return &AnalyticsHandler{eventRepo: eventRepo}
}

// HandleGetLoginMetrics handles GET /analytics/login-metrics.
func (h *AnalyticsHandler) HandleGetLoginMetrics(w http.ResponseWriter, r *http.Request) {
	tenantID := GetTenantID(r.Context())
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "missing_tenant", "X-Tenant-ID header required")
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	metrics, err := h.eventRepo.GetLoginMetrics(r.Context(), tenantID, period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, metrics)
}

// HandleGetSuspiciousIPs handles GET /analytics/suspicious-ips.
func (h *AnalyticsHandler) HandleGetSuspiciousIPs(w http.ResponseWriter, r *http.Request) {
	tenantID := GetTenantID(r.Context())
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "missing_tenant", "X-Tenant-ID header required")
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	ips, err := h.eventRepo.GetSuspiciousIPs(r.Context(), tenantID, limit, period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ips)
}
