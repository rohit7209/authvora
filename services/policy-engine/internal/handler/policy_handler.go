package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/authvora/policy-engine/internal/model"
	"github.com/authvora/policy-engine/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// PolicyHandler provides HTTP handlers for policy endpoints.
type PolicyHandler struct {
	svc *service.PolicyService
}

// NewPolicyHandler creates a new PolicyHandler.
func NewPolicyHandler(svc *service.PolicyService) *PolicyHandler {
	return &PolicyHandler{svc: svc}
}

// HandleGetPolicy handles GET /policies/:tenantId
func (h *PolicyHandler) HandleGetPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	if tenantID == "" || uuid.Validate(tenantID) != nil {
		writeError(w, "invalid_request", "tenantId is required and must be a valid UUID", http.StatusBadRequest)
		return
	}

	policy, err := h.svc.GetPolicy(r.Context(), tenantID)
	if err != nil {
		writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
		return
	}

	resp := policyToResponse(policy)
	writeJSON(w, http.StatusOK, resp)
}

// HandleUpdatePolicy handles PUT /policies/:tenantId
func (h *PolicyHandler) HandleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	if tenantID == "" || uuid.Validate(tenantID) != nil {
		writeError(w, "invalid_request", "tenantId is required and must be a valid UUID", http.StatusBadRequest)
		return
	}

	var req model.UpdatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "invalid JSON body", http.StatusBadRequest)
		return
	}

	policy, err := h.svc.UpdatePolicy(r.Context(), tenantID, req)
	if err != nil {
		writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
		return
	}

	resp := policyToResponse(policy)
	writeJSON(w, http.StatusOK, resp)
}

// HandleValidatePassword handles POST /policies/validate-password
func (h *PolicyHandler) HandleValidatePassword(w http.ResponseWriter, r *http.Request) {
	var req model.ValidatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.TenantID == "" || uuid.Validate(req.TenantID) != nil {
		writeError(w, "invalid_request", "tenant_id is required and must be a valid UUID", http.StatusBadRequest)
		return
	}

	resp, err := h.svc.ValidatePassword(r.Context(), req.TenantID, req.Password)
	if err != nil {
		writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleEvaluateMFA handles GET /policies/:tenantId/evaluate-mfa?risk_score=42
func (h *PolicyHandler) HandleEvaluateMFA(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	if tenantID == "" || uuid.Validate(tenantID) != nil {
		writeError(w, "invalid_request", "tenantId is required and must be a valid UUID", http.StatusBadRequest)
		return
	}

	riskScoreStr := r.URL.Query().Get("risk_score")
	if riskScoreStr == "" {
		writeError(w, "invalid_request", "risk_score query parameter is required", http.StatusBadRequest)
		return
	}

	riskScore, err := strconv.Atoi(riskScoreStr)
	if err != nil || riskScore < 0 {
		writeError(w, "invalid_request", "risk_score must be a non-negative integer", http.StatusBadRequest)
		return
	}

	required, err := h.svc.EvaluateMFARequirement(r.Context(), tenantID, riskScore)
	if err != nil {
		writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"mfa_required": required})
}

func policyToResponse(p *model.TenantPolicy) model.PolicyResponse {
	return model.PolicyResponse{
		ID:                     p.ID,
		TenantID:               p.TenantID,
		PasswordMinLength:      p.PasswordMinLength,
		PasswordRequireUpper:   p.PasswordRequireUpper,
		PasswordRequireNumber:  p.PasswordRequireNumber,
		PasswordRequireSpecial: p.PasswordRequireSpecial,
		MFARequired:            p.MFARequired,
		MFARiskThreshold:       p.MFARiskThreshold,
		SessionMaxAgeHours:     p.SessionMaxAgeHours,
		MaxSessionsPerUser:     p.MaxSessionsPerUser,
		RateLimitPerMinute:     p.RateLimitPerMinute,
		LoginMaxAttempts:       p.LoginMaxAttempts,
		LockoutDurationMinutes: p.LockoutDurationMinutes,
		AllowedOrigins:         p.AllowedOrigins,
		CreatedAt:              p.CreatedAt,
		UpdatedAt:              p.UpdatedAt,
	}
}

func writeError(w http.ResponseWriter, code, message string, status int) {
	resp := model.ErrorResponse{}
	resp.Error.Code = code
	resp.Error.Message = message
	resp.Error.Status = status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
