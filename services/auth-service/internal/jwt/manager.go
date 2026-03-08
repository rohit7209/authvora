package jwt

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/authvora/auth-service/internal/model"
	"github.com/authvora/auth-service/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	accessTokenExpiry = 15 * time.Minute
)

// Manager handles JWT generation and JWKS.
type Manager struct {
	issuer       string
	signingRepo  *repository.SigningKeyRepository
}

// NewManager creates a new JWT Manager.
func NewManager(issuer string, signingRepo *repository.SigningKeyRepository) *Manager {
	return &Manager{
		issuer:      issuer,
		signingRepo: signingRepo,
	}
}

// Claims represents JWT claims.
type Claims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	TID   string `json:"tid"`
}

// JWKS represents the JSON Web Key Set structure.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key.
type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// GenerateAccessToken creates a signed JWT access token.
func (m *Manager) GenerateAccessToken(ctx context.Context, user *model.User, tenantID, tenantSlug string) (string, error) {
	if err := m.EnsureSigningKey(ctx, tenantID); err != nil {
		return "", err
	}

	key, err := m.signingRepo.GetActiveSigningKey(ctx, tenantID)
	if err != nil {
		return "", err
	}

	privateKey, err := parseRSAPrivateKey(key.PrivateKey)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{tenantSlug},
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		Email: user.Email,
		TID:   tenantID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = key.KeyID

	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", err
	}

	return signed, nil
}

// GenerateRefreshToken creates a random refresh token and returns raw token and hash.
func (m *Manager) GenerateRefreshToken() (rawToken, tokenHash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	rawToken = base64.URLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash = base64.URLEncoding.EncodeToString(hash[:])
	return rawToken, tokenHash, nil
}

// GetJWKS returns the JWKS for a tenant.
func (m *Manager) GetJWKS(ctx context.Context, tenantID string) (*JWKS, error) {
	keys, err := m.signingRepo.GetAllActiveKeys(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	jwks := &JWKS{Keys: make([]JWK, 0, len(keys))}
	for _, k := range keys {
		pubKey, err := parseRSAPublicKey(k.PublicKey)
		if err != nil {
			continue
		}
		n, e := encodeRSAPublicKey(pubKey)
		jwks.Keys = append(jwks.Keys, JWK{
			Kty: "RSA",
			Kid: k.KeyID,
			Use: "sig",
			Alg: "RS256",
			N:   n,
			E:   e,
		})
	}

	return jwks, nil
}

// EnsureSigningKey creates an RSA key pair if none exists for the tenant.
func (m *Manager) EnsureSigningKey(ctx context.Context, tenantID string) error {
	_, err := m.signingRepo.GetActiveSigningKey(ctx, tenantID)
	if err == nil {
		return nil
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	pubPEM, privPEM, err := encodeKeyPair(privateKey)
	if err != nil {
		return err
	}

	key := &model.SigningKey{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		KeyID:      uuid.New().String(),
		PublicKey:  pubPEM,
		PrivateKey: privPEM,
		Active:     true,
		CreatedAt:  time.Now(),
	}

	return m.signingRepo.CreateSigningKey(ctx, key)
}

// ToJWKSJSON returns the JWKS as JSON bytes.
func (m *Manager) ToJWKSJSON(ctx context.Context, tenantID string) ([]byte, error) {
	jwks, err := m.GetJWKS(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(jwks)
}
