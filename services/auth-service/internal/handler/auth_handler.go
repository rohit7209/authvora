package handler

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/authvora/auth-service/internal/jwt"
	"github.com/authvora/auth-service/internal/model"
	"github.com/authvora/auth-service/internal/service"
)

// AuthHandler handles authentication HTTP requests.
type AuthHandler struct {
	authService *service.AuthService
	jwtManager  *jwt.Manager
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *service.AuthService, jwtManager *jwt.Manager) *AuthHandler {
	return &AuthHandler{authService: authService, jwtManager: jwtManager}
}

// HandleRegister handles POST /register.
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	tenantID := GetTenantID(r.Context())
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "missing_tenant", "X-Tenant-ID header required")
		return
	}

	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	resp, err := h.authService.Register(r.Context(), tenantID, req)
	if err != nil {
		switch {
		case err == service.ErrEmailExists:
			writeError(w, http.StatusConflict, "email_exists", "Email already registered")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// HandleLogin handles POST /login.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	tenantID := GetTenantID(r.Context())
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "missing_tenant", "X-Tenant-ID header required")
		return
	}

	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	ipAddress := extractIP(r)

	userAgent := r.Header.Get("User-Agent")

	resp, err := h.authService.Login(r.Context(), tenantID, req, ipAddress, userAgent)
	if err != nil {
		switch {
		case err == service.ErrInvalidCredentials:
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleOAuthGoogle handles POST /oauth/google.
func (h *AuthHandler) HandleOAuthGoogle(w http.ResponseWriter, r *http.Request) {
	tenantID := GetTenantID(r.Context())
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "missing_tenant", "X-Tenant-ID header required")
		return
	}

	var req model.OAuthGoogleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	resp, err := h.authService.OAuthGoogle(r.Context(), tenantID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "oauth_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleRefreshToken handles POST /refresh.
func (h *AuthHandler) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	resp, err := h.authService.RefreshToken(r.Context(), req)
	if err != nil {
		switch {
		case err == service.ErrInvalidToken:
			writeError(w, http.StatusUnauthorized, "invalid_token", "Invalid or expired refresh token")
		case err == service.ErrTokenReplay:
			writeError(w, http.StatusUnauthorized, "token_replay", "Refresh token reuse detected")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleRevokeToken handles POST /revoke.
func (h *AuthHandler) HandleRevokeToken(w http.ResponseWriter, r *http.Request) {
	var req model.RevokeTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if err := h.authService.RevokeToken(r.Context(), req.RefreshToken); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleJWKS handles GET /jwks.
func (h *AuthHandler) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	tenantID := GetTenantID(r.Context())
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "missing_tenant", "X-Tenant-ID header required")
		return
	}

	jwks, err := h.jwtManager.GetJWKS(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jwks)
}

func extractIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
