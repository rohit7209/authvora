package handler

import (
	"net/http"

	"github.com/authvora/auth-service/internal/model"
	"github.com/authvora/auth-service/internal/service"
	"github.com/go-chi/chi/v5"
)

// UserHandler handles user HTTP requests.
type UserHandler struct {
	authService *service.AuthService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(authService *service.AuthService) *UserHandler {
	return &UserHandler{authService: authService}
}

// HandleGetCurrentUser handles GET /me.
func (h *UserHandler) HandleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	tenantID := GetTenantID(r.Context())
	userID := GetUserID(r.Context())
	if tenantID == "" || userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), tenantID, userID)
	if err != nil {
		switch {
		case err == service.ErrUserNotFound:
			writeError(w, http.StatusNotFound, "user_not_found", "User not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// HandleGetUserByID handles GET /users/{id}.
func (h *UserHandler) HandleGetUserByID(w http.ResponseWriter, r *http.Request) {
	tenantID := GetTenantID(r.Context())
	userID := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" {
		writeError(w, http.StatusBadRequest, "missing_params", "Tenant and user ID required")
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), tenantID, userID)
	if err != nil {
		switch {
		case err == service.ErrUserNotFound:
			writeError(w, http.StatusNotFound, "user_not_found", "User not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// toUserResponse converts User to UserResponse.
func toUserResponse(user *model.User) *model.UserResponse {
	if user == nil {
		return nil
	}
	return &model.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
	}
}
