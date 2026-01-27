package auth

import (
	"errors"
	"time"

	"github.com/erp/backend/internal/infrastructure/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType represents the type of JWT token
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// Common errors
var (
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("token has expired")
	ErrInvalidTokenType   = errors.New("invalid token type")
	ErrInvalidClaims      = errors.New("invalid token claims")
	ErrTokenNotYetValid   = errors.New("token is not yet valid")
	ErrMissingTenantID    = errors.New("missing tenant_id in claims")
	ErrMissingUserID      = errors.New("missing user_id in claims")
	ErrMaxRefreshExceeded = errors.New("maximum refresh count exceeded")
	ErrTokenBlacklisted   = errors.New("token has been revoked")
)

// Claims represents custom JWT claims
type Claims struct {
	jwt.RegisteredClaims
	TenantID     string    `json:"tenant_id"`
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	RoleIDs      []string  `json:"role_ids,omitempty"`
	Permissions  []string  `json:"permissions,omitempty"`
	TokenType    TokenType `json:"token_type"`
	RefreshCount int       `json:"refresh_count,omitempty"`
}

// TokenPair represents an access and refresh token pair
type TokenPair struct {
	AccessToken           string    `json:"access_token"`
	RefreshToken          string    `json:"refresh_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	TokenType             string    `json:"token_type"` // Bearer
}

// JWTService handles JWT token operations
type JWTService struct {
	accessSecret      []byte
	refreshSecret     []byte
	accessExpiration  time.Duration
	refreshExpiration time.Duration
	issuer            string
	maxRefreshCount   int
}

// NewJWTService creates a new JWT service
func NewJWTService(cfg config.JWTConfig) *JWTService {
	refreshSecret := []byte(cfg.RefreshSecret)
	if cfg.RefreshSecret == "" {
		refreshSecret = []byte(cfg.Secret)
	}

	return &JWTService{
		accessSecret:      []byte(cfg.Secret),
		refreshSecret:     refreshSecret,
		accessExpiration:  cfg.AccessTokenExpiration,
		refreshExpiration: cfg.RefreshTokenExpiration,
		issuer:            cfg.Issuer,
		maxRefreshCount:   cfg.MaxRefreshCount,
	}
}

// GenerateTokenInput contains input for token generation
type GenerateTokenInput struct {
	TenantID    uuid.UUID
	UserID      uuid.UUID
	Username    string
	RoleIDs     []uuid.UUID
	Permissions []string
}

// GenerateTokenPair generates both access and refresh tokens
func (s *JWTService) GenerateTokenPair(input GenerateTokenInput) (*TokenPair, error) {
	now := time.Now()
	jti := uuid.New().String()

	// Convert UUIDs to strings
	roleIDStrings := make([]string, len(input.RoleIDs))
	for i, rid := range input.RoleIDs {
		roleIDStrings[i] = rid.String()
	}

	// Generate access token
	accessClaims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    s.issuer,
			Subject:   input.UserID.String(),
			Audience:  jwt.ClaimStrings{s.issuer},
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExpiration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		TenantID:    input.TenantID.String(),
		UserID:      input.UserID.String(),
		Username:    input.Username,
		RoleIDs:     roleIDStrings,
		Permissions: input.Permissions,
		TokenType:   TokenTypeAccess,
	}

	accessToken, err := s.generateToken(accessClaims, s.accessSecret)
	if err != nil {
		return nil, err
	}

	// Generate refresh token (with minimal claims for security)
	refreshClaims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    s.issuer,
			Subject:   input.UserID.String(),
			Audience:  jwt.ClaimStrings{s.issuer},
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshExpiration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		TenantID:     input.TenantID.String(),
		UserID:       input.UserID.String(),
		TokenType:    TokenTypeRefresh,
		RefreshCount: 0,
	}

	refreshToken, err := s.generateToken(refreshClaims, s.refreshSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  now.Add(s.accessExpiration),
		RefreshTokenExpiresAt: now.Add(s.refreshExpiration),
		TokenType:             "Bearer",
	}, nil
}

// generateToken creates a signed JWT token
func (s *JWTService) generateToken(claims *Claims, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ValidateAccessToken validates an access token and returns its claims
func (s *JWTService) ValidateAccessToken(tokenString string) (*Claims, error) {
	return s.validateToken(tokenString, s.accessSecret, TokenTypeAccess)
}

// ValidateRefreshToken validates a refresh token and returns its claims
func (s *JWTService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return s.validateToken(tokenString, s.refreshSecret, TokenTypeRefresh)
}

// validateToken validates a JWT token
func (s *JWTService) validateToken(tokenString string, secret []byte, expectedType TokenType) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, ErrTokenNotYetValid
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	// Validate token type
	if claims.TokenType != expectedType {
		return nil, ErrInvalidTokenType
	}

	// Validate required claims
	if claims.TenantID == "" {
		return nil, ErrMissingTenantID
	}
	if claims.UserID == "" {
		return nil, ErrMissingUserID
	}

	return claims, nil
}

// RefreshTokenPair refreshes tokens using a valid refresh token
func (s *JWTService) RefreshTokenPair(refreshToken string, permissions []string) (*TokenPair, error) {
	claims, err := s.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Check refresh count limit
	if claims.RefreshCount >= s.maxRefreshCount {
		return nil, ErrMaxRefreshExceeded
	}

	// Parse UUIDs from claims
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return nil, ErrInvalidClaims
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, ErrInvalidClaims
	}

	// Generate new token pair
	now := time.Now()
	jti := uuid.New().String()

	// Generate new access token
	accessClaims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    s.issuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{s.issuer},
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExpiration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		TenantID:    tenantID.String(),
		UserID:      userID.String(),
		Username:    claims.Username,
		RoleIDs:     claims.RoleIDs,
		Permissions: permissions,
		TokenType:   TokenTypeAccess,
	}

	accessToken, err := s.generateToken(accessClaims, s.accessSecret)
	if err != nil {
		return nil, err
	}

	// Generate new refresh token with incremented count
	refreshClaims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    s.issuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{s.issuer},
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshExpiration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		TenantID:     tenantID.String(),
		UserID:       userID.String(),
		TokenType:    TokenTypeRefresh,
		RefreshCount: claims.RefreshCount + 1,
	}

	newRefreshToken, err := s.generateToken(refreshClaims, s.refreshSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:           accessToken,
		RefreshToken:          newRefreshToken,
		AccessTokenExpiresAt:  now.Add(s.accessExpiration),
		RefreshTokenExpiresAt: now.Add(s.refreshExpiration),
		TokenType:             "Bearer",
	}, nil
}

// GetTenantUUID extracts and parses the tenant ID from claims
func (c *Claims) GetTenantUUID() (uuid.UUID, error) {
	return uuid.Parse(c.TenantID)
}

// GetUserUUID extracts and parses the user ID from claims
func (c *Claims) GetUserUUID() (uuid.UUID, error) {
	return uuid.Parse(c.UserID)
}

// GetRoleUUIDs extracts and parses the role IDs from claims
func (c *Claims) GetRoleUUIDs() ([]uuid.UUID, error) {
	roleIDs := make([]uuid.UUID, 0, len(c.RoleIDs))
	for _, rid := range c.RoleIDs {
		id, err := uuid.Parse(rid)
		if err != nil {
			return nil, err
		}
		roleIDs = append(roleIDs, id)
	}
	return roleIDs, nil
}

// HasPermission checks if the claims contain a specific permission
func (c *Claims) HasPermission(permission string) bool {
	for _, p := range c.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasAnyPermission checks if the claims contain any of the specified permissions
func (c *Claims) HasAnyPermission(permissions ...string) bool {
	for _, required := range permissions {
		if c.HasPermission(required) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if the claims contain all of the specified permissions
func (c *Claims) HasAllPermissions(permissions ...string) bool {
	for _, required := range permissions {
		if !c.HasPermission(required) {
			return false
		}
	}
	return true
}

// GetIssuedAtTime returns the token's issued-at time as time.Time
func (c *Claims) GetIssuedAtTime() time.Time {
	if c.IssuedAt != nil {
		return c.IssuedAt.Time
	}
	return time.Time{}
}

// GetExpiresAtTime returns the token's expiration time as time.Time
func (c *Claims) GetExpiresAtTime() time.Time {
	if c.ExpiresAt != nil {
		return c.ExpiresAt.Time
	}
	return time.Time{}
}

// GetRemainingTTL returns the remaining time until the token expires
func (c *Claims) GetRemainingTTL() time.Duration {
	if c.ExpiresAt == nil {
		return 0
	}
	remaining := time.Until(c.ExpiresAt.Time)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetAccessTokenExpiration returns the access token expiration duration
func (s *JWTService) GetAccessTokenExpiration() time.Duration {
	return s.accessExpiration
}

// GetRefreshTokenExpiration returns the refresh token expiration duration
func (s *JWTService) GetRefreshTokenExpiration() time.Duration {
	return s.refreshExpiration
}
