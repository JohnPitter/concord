package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenExpiry  = 15 * time.Minute
	RefreshTokenExpiry = 30 * 24 * time.Hour // 30 days
)

// Claims represents the JWT payload for Concord tokens.
type Claims struct {
	UserID   string `json:"uid"`
	GitHubID int64  `json:"gid"`
	Username string `json:"usr"`
	jwt.RegisteredClaims
}

// TokenPair holds an access token and a refresh token.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// JWTManager handles JWT token generation and validation.
type JWTManager struct {
	secret []byte
}

// NewJWTManager creates a new JWT manager with the given secret.
// The secret must be at least 32 bytes for HS256.
func NewJWTManager(secret string) (*JWTManager, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT secret must be at least 32 characters, got %d", len(secret))
	}
	return &JWTManager{secret: []byte(secret)}, nil
}

// GenerateTokenPair creates a new access + refresh token pair.
// Complexity: O(1)
func (j *JWTManager) GenerateTokenPair(userID string, githubID int64, username string) (*TokenPair, error) {
	now := time.Now()

	accessClaims := Claims{
		UserID:   userID,
		GitHubID: githubID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "concord",
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(j.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshClaims := Claims{
		UserID:   userID,
		GitHubID: githubID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "concord-refresh",
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString(j.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresAt:    now.Add(AccessTokenExpiry).Unix(),
	}, nil
}

// ValidateToken parses and validates a JWT token string.
// Complexity: O(1)
func (j *JWTManager) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// RefreshAccessToken generates a new access token from a valid refresh token.
// Complexity: O(1)
func (j *JWTManager) RefreshAccessToken(refreshTokenStr string) (*TokenPair, error) {
	claims, err := j.ValidateToken(refreshTokenStr)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.Issuer != "concord-refresh" {
		return nil, fmt.Errorf("token is not a refresh token")
	}

	return j.GenerateTokenPair(claims.UserID, claims.GitHubID, claims.Username)
}
