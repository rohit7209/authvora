package model

// RegisterRequest is the request body for user registration.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// LoginRequest is the request body for login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// OAuthGoogleRequest is the request body for Google OAuth.
type OAuthGoogleRequest struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
}

// RefreshTokenRequest is the request body for token refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RevokeTokenRequest is the request body for token revocation.
type RevokeTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// MFASetupRequest is the request body for MFA setup.
type MFASetupRequest struct {
	// Reserved for future MFA setup
}

// MFAVerifyRequest is the request body for MFA verification.
type MFAVerifyRequest struct {
	Code string `json:"code"`
}

// AuthResponse is the response for auth operations.
type AuthResponse struct {
	User         *UserResponse `json:"user"`
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    int           `json:"expires_in"`
}

// UserResponse is the user data in API responses.
type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// ErrorResponse is the standard error response.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}
