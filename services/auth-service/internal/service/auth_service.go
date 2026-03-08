package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/authvora/auth-service/internal/jwt"
	"github.com/authvora/auth-service/internal/model"
	"github.com/authvora/auth-service/internal/oauth"
	"github.com/authvora/auth-service/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"
)

const (
	argon2Memory  = 64 * 1024
	argon2Time    = 3
	argon2Threads = 2
	argon2KeyLen  = 32
	saltLen       = 16
)

var (
	ErrEmailExists      = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidToken     = errors.New("invalid or expired refresh token")
	ErrTokenReplay      = errors.New("refresh token reuse detected")
)

// AuthService orchestrates authentication flows.
type AuthService struct {
	userRepo       *repository.UserRepository
	credRepo       *repository.CredentialRepository
	sessionRepo    *repository.SessionRepository
	refreshRepo    *repository.RefreshTokenRepository
	eventRepo      *repository.EventRepository
	oauthRepo      *repository.OAuthConnectionRepository
	tenantRepo     *repository.TenantRepository
	jwtManager     *jwt.Manager
	googleOAuth    *oauth.GoogleOAuth
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo *repository.UserRepository,
	credRepo *repository.CredentialRepository,
	sessionRepo *repository.SessionRepository,
	refreshRepo *repository.RefreshTokenRepository,
	eventRepo *repository.EventRepository,
	oauthRepo *repository.OAuthConnectionRepository,
	tenantRepo *repository.TenantRepository,
	jwtManager *jwt.Manager,
	googleOAuth *oauth.GoogleOAuth,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		credRepo:    credRepo,
		sessionRepo: sessionRepo,
		refreshRepo: refreshRepo,
		eventRepo:   eventRepo,
		oauthRepo:   oauthRepo,
		tenantRepo:  tenantRepo,
		jwtManager:  jwtManager,
		googleOAuth: googleOAuth,
	}
}

// Register creates a new user and returns auth tokens.
func (s *AuthService) Register(ctx context.Context, tenantID string, req model.RegisterRequest) (*model.AuthResponse, error) {
	if err := validateRegisterRequest(&req); err != nil {
		return nil, err
	}

	existing, _ := s.userRepo.GetUserByEmail(ctx, tenantID, strings.ToLower(req.Email))
	if existing != nil {
		return nil, ErrEmailExists
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.CreateUser(ctx, tenantID, strings.ToLower(req.Email), req.Name)
	if err != nil {
		return nil, err
	}

	if err := s.credRepo.CreateCredential(ctx, user.ID, tenantID, hash); err != nil {
		return nil, err
	}

	return s.issueAuthResponse(ctx, user, tenantID, "", "")
}

// Login authenticates a user and returns auth tokens.
func (s *AuthService) Login(ctx context.Context, tenantID string, req model.LoginRequest, ipAddress, userAgent string) (*model.AuthResponse, error) {
	if err := validateLoginRequest(&req); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetUserByEmail(ctx, tenantID, strings.ToLower(req.Email))
	if err != nil || user == nil {
		s.logLoginEvent(ctx, tenantID, nil, req.Email, false, "user_not_found", ipAddress, userAgent)
		return nil, ErrInvalidCredentials
	}

	cred, err := s.credRepo.GetCredentialByUserID(ctx, user.ID)
	if err != nil || cred == nil {
		s.logLoginEvent(ctx, tenantID, &user.ID, req.Email, false, "no_credential", ipAddress, userAgent)
		return nil, ErrInvalidCredentials
	}

	if !verifyPassword(req.Password, cred.PasswordHash) {
		s.logLoginEvent(ctx, tenantID, &user.ID, req.Email, false, "invalid_password", ipAddress, userAgent)
		return nil, ErrInvalidCredentials
	}

	s.logLoginEvent(ctx, tenantID, &user.ID, req.Email, true, "", ipAddress, userAgent)
	return s.issueAuthResponse(ctx, user, tenantID, ipAddress, userAgent)
}

// OAuthGoogle exchanges a Google code and returns auth tokens.
func (s *AuthService) OAuthGoogle(ctx context.Context, tenantID string, req model.OAuthGoogleRequest) (*model.AuthResponse, error) {
	if req.Code == "" || req.RedirectURI == "" {
		return nil, errors.New("code and redirect_uri required")
	}

	tenant, err := s.tenantRepo.GetTenantByID(ctx, tenantID)
	if err != nil || tenant == nil {
		return nil, errors.New("tenant not found")
	}

	info, err := s.googleOAuth.ExchangeCode(ctx, req.Code, req.RedirectURI, tenant)
	if err != nil {
		return nil, err
	}

	conn, err := s.oauthRepo.GetOAuthConnection(ctx, tenantID, "google", info.Sub)
	if err == nil && conn != nil {
		user, err := s.userRepo.GetUserByID(ctx, tenantID, conn.UserID)
		if err != nil {
			return nil, err
		}
		return s.issueAuthResponse(ctx, user, tenantID, "", "")
	}

	user, err := s.userRepo.GetUserByEmail(ctx, tenantID, strings.ToLower(info.Email))
	if err == nil && user != nil {
		oauthConn := &model.OAuthConnection{
			TenantID:    tenantID,
			UserID:      user.ID,
			Provider:    "google",
			ProviderUID: info.Sub,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := s.oauthRepo.CreateOAuthConnection(ctx, oauthConn); err != nil {
			return nil, err
		}
		return s.issueAuthResponse(ctx, user, tenantID, "", "")
	}

	user, err = s.userRepo.CreateUser(ctx, tenantID, strings.ToLower(info.Email), info.Name)
	if err != nil {
		return nil, err
	}
	user.AvatarURL = info.AvatarURL

	oauthConn := &model.OAuthConnection{
		TenantID:    tenantID,
		UserID:      user.ID,
		Provider:    "google",
		ProviderUID: info.Sub,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.oauthRepo.CreateOAuthConnection(ctx, oauthConn); err != nil {
		return nil, err
	}

	s.logLoginEvent(ctx, tenantID, &user.ID, user.Email, true, "", "", "")
	return s.issueAuthResponse(ctx, user, tenantID, "", "")
}

// RefreshToken exchanges a refresh token for new tokens.
func (s *AuthService) RefreshToken(ctx context.Context, req model.RefreshTokenRequest) (*model.AuthResponse, error) {
	if req.RefreshToken == "" {
		return nil, ErrInvalidToken
	}

	hash := hashRefreshToken(req.RefreshToken)
	token, err := s.refreshRepo.GetRefreshTokenByHash(ctx, hash)
	if err != nil || token == nil {
		return nil, ErrInvalidToken
	}

	if token.Used || token.Revoked {
		_ = s.refreshRepo.RevokeRefreshTokenFamily(ctx, token.FamilyID)
		return nil, ErrTokenReplay
	}

	if time.Now().After(token.ExpiresAt) {
		return nil, ErrInvalidToken
	}

	if err := s.refreshRepo.MarkRefreshTokenUsed(ctx, token.ID); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetUserByID(ctx, token.TenantID, token.UserID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	tenant, err := s.tenantRepo.GetTenantByID(ctx, token.TenantID)
	if err != nil || tenant == nil {
		return nil, errors.New("tenant not found")
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(ctx, user, token.TenantID, tenant.Slug)
	if err != nil {
		return nil, err
	}

	rawRefresh, refreshHash, err := s.jwtManager.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	newToken := &model.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TenantID:  token.TenantID,
		SessionID: token.SessionID,
		FamilyID:  token.FamilyID,
		TokenHash: refreshHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := s.refreshRepo.CreateRefreshToken(ctx, newToken); err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		User:         toUserResponse(user),
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    900,
	}, nil
}

// RevokeToken revokes a refresh token.
func (s *AuthService) RevokeToken(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return ErrInvalidToken
	}
	hash := hashRefreshToken(refreshToken)
	token, err := s.refreshRepo.GetRefreshTokenByHash(ctx, hash)
	if err != nil || token == nil {
		return nil
	}
	return s.refreshRepo.RevokeRefreshTokenFamily(ctx, token.FamilyID)
}

// GetUserByID returns a user by ID.
func (s *AuthService) GetUserByID(ctx context.Context, tenantID, userID string) (*model.User, error) {
	user, err := s.userRepo.GetUserByID(ctx, tenantID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *AuthService) issueAuthResponse(ctx context.Context, user *model.User, tenantID, ipAddress, userAgent string) (*model.AuthResponse, error) {
	tenant, err := s.tenantRepo.GetTenantByID(ctx, tenantID)
	if err != nil || tenant == nil {
		return nil, errors.New("tenant not found")
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(ctx, user, tenantID, tenant.Slug)
	if err != nil {
		return nil, err
	}

	rawRefresh, refreshHash, err := s.jwtManager.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	sessionID := uuid.New().String()
	familyID := uuid.New().String()

	session := &model.Session{
		ID:        sessionID,
		UserID:    user.ID,
		TenantID:  tenantID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := s.sessionRepo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	refreshToken := &model.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TenantID:  tenantID,
		SessionID: sessionID,
		FamilyID:  familyID,
		TokenHash: refreshHash,
		ExpiresAt: session.ExpiresAt,
		CreatedAt: time.Now(),
	}
	if err := s.refreshRepo.CreateRefreshToken(ctx, refreshToken); err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		User:         toUserResponse(user),
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    900,
	}, nil
}

func (s *AuthService) logLoginEvent(ctx context.Context, tenantID string, userID *string, email string, success bool, failureReason, ipAddress, userAgent string) {
	eventType := "login_success"
	metadata := fmt.Sprintf(`{"email":%q}`, email)
	if !success {
		eventType = "login_failure"
		metadata = fmt.Sprintf(`{"email":%q,"failure_reason":%q}`, email, failureReason)
	}
	event := &model.LoginEvent{
		TenantID:  tenantID,
		UserID:    userID,
		EventType: eventType,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
	_ = s.eventRepo.CreateLoginEvent(ctx, event)
}

func validateRegisterRequest(req *model.RegisterRequest) error {
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return errors.New("email, password, and name required")
	}
	if len(req.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func validateLoginRequest(req *model.LoginRequest) error {
	if req.Email == "" || req.Password == "" {
		return errors.New("email and password required")
	}
	return nil
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory, argon2Time, argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func verifyPassword(password, storedHash string) bool {
	parts := strings.Split(storedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var m, t, p uint32
	_, _ = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p)
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	computed := argon2.IDKey([]byte(password), salt, t, m, uint8(p), uint32(len(hash)))
	return len(computed) == len(hash) && subtleConstantTimeCompare(computed, hash)
}

func subtleConstantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

func hashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(h[:])
}

func toUserResponse(u *model.User) *model.UserResponse {
	return &model.UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
	}
}
