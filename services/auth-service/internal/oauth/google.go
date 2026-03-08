package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/authvora/auth-service/internal/model"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleUserInfo holds user info from Google.
type GoogleUserInfo struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	AvatarURL string `json:"picture"`
	Sub      string `json:"sub"`
}

// GoogleOAuth handles Google OAuth flows.
type GoogleOAuth struct {
	clientID     string
	clientSecret string
}

// NewGoogleOAuth creates a new GoogleOAuth client.
func NewGoogleOAuth(clientID, clientSecret string) *GoogleOAuth {
	return &GoogleOAuth{
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

// ExchangeCode exchanges an authorization code for tokens and fetches user info.
func (g *GoogleOAuth) ExchangeCode(ctx context.Context, code, redirectURI string, tenant *model.Tenant) (*GoogleUserInfo, error) {
	config := &oauth2.Config{
		ClientID:     g.clientID,
		ClientSecret: g.clientSecret,
		RedirectURL:  redirectURI,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	client := config.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo API error %d: %s", resp.StatusCode, string(body))
	}

	var info GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode userinfo: %w", err)
	}

	return &info, nil
}

// GetAuthURL returns the Google OAuth authorization URL.
func (g *GoogleOAuth) GetAuthURL(redirectURI, state string) string {
	config := &oauth2.Config{
		ClientID:     g.clientID,
		ClientSecret: g.clientSecret,
		RedirectURL:  redirectURI,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
	return config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}
