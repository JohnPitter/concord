package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/concord-chat/concord/internal/security"
)

// AuthState represents the current authentication state exposed to the frontend.
type AuthState struct {
	Authenticated bool   `json:"authenticated"`
	User          *User  `json:"user,omitempty"`
	AccessToken   string `json:"access_token,omitempty"`
	ExpiresAt     int64  `json:"expires_at,omitempty"`
}

// Service orchestrates the authentication flow.
type Service struct {
	github     *GitHubOAuth
	jwt        *JWTManager
	repo       *Repository
	crypto     *security.CryptoManager
	encryptKey []byte // 32-byte key derived from JWT secret for encrypting refresh tokens
	logger     zerolog.Logger
}

// NewService creates a new auth service.
// The encryptKey is a 32-byte key used to encrypt refresh tokens at rest.
func NewService(github *GitHubOAuth, jwt *JWTManager, repo *Repository, crypto *security.CryptoManager, encryptKey []byte, logger zerolog.Logger) *Service {
	return &Service{
		github:     github,
		jwt:        jwt,
		repo:       repo,
		crypto:     crypto,
		encryptKey: encryptKey,
		logger:     logger.With().Str("component", "auth_service").Logger(),
	}
}

// StartLogin initiates the GitHub Device Flow.
// Returns the device code response for the frontend to display.
func (s *Service) StartLogin(ctx context.Context) (*DeviceCodeResponse, error) {
	s.logger.Info().Msg("starting login flow")
	return s.github.RequestDeviceCode(ctx)
}

// CompleteLogin polls for the token and creates the local user + session.
func (s *Service) CompleteLogin(ctx context.Context, deviceCode string, interval int) (*AuthState, error) {
	s.logger.Info().Msg("completing login flow")

	// Poll for access token
	accessToken, err := s.github.PollForToken(ctx, deviceCode, interval)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Fetch GitHub user profile
	ghUser, err := s.github.FetchUser(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user profile: %w", err)
	}

	// Generate user ID (deterministic from GitHub ID)
	userID := fmt.Sprintf("gh_%d", ghUser.ID)

	displayName := ghUser.Name
	if displayName == "" {
		displayName = ghUser.Login
	}

	// Upsert user in local DB
	user := &User{
		ID:          userID,
		GitHubID:    ghUser.ID,
		Username:    ghUser.Login,
		DisplayName: displayName,
		AvatarURL:   ghUser.AvatarURL,
	}

	if err := s.repo.UpsertUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	// Generate JWT token pair
	tokenPair, err := s.jwt.GenerateTokenPair(userID, ghUser.ID, ghUser.Login)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Encrypt refresh token at rest (AES-256-GCM)
	encryptedRefresh, err := s.crypto.EncryptAES([]byte(tokenPair.RefreshToken), s.encryptKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	tokenHash := sha256.Sum256([]byte(tokenPair.RefreshToken))

	session := &Session{
		ID:               uuid.New().String(),
		UserID:           userID,
		RefreshTokenHash: hex.EncodeToString(tokenHash[:]),
		EncryptedRefresh: base64.StdEncoding.EncodeToString(encryptedRefresh),
		ExpiresAt:        time.Now().Add(RefreshTokenExpiry),
	}

	if err := s.repo.SaveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	s.logger.Info().
		Str("user_id", userID).
		Str("username", ghUser.Login).
		Msg("login completed successfully")

	return &AuthState{
		Authenticated: true,
		User:          user,
		AccessToken:   tokenPair.AccessToken,
		ExpiresAt:     tokenPair.ExpiresAt,
	}, nil
}

// RestoreSession attempts to restore a session from an encrypted refresh token.
func (s *Service) RestoreSession(ctx context.Context, userID string) (*AuthState, error) {
	s.logger.Info().Str("user_id", userID).Msg("attempting to restore session")

	session, err := s.repo.GetActiveSession(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return &AuthState{Authenticated: false}, nil
	}

	// Decode and decrypt refresh token
	encryptedBytes, err := base64.StdEncoding.DecodeString(session.EncryptedRefresh)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to decode refresh token, clearing session")
		s.repo.DeleteUserSessions(ctx, userID)
		return &AuthState{Authenticated: false}, nil
	}

	refreshTokenBytes, err := s.crypto.DecryptAES(encryptedBytes, s.encryptKey)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to decrypt refresh token, clearing session")
		s.repo.DeleteUserSessions(ctx, userID)
		return &AuthState{Authenticated: false}, nil
	}

	// Generate new token pair from refresh token
	tokenPair, err := s.jwt.RefreshAccessToken(string(refreshTokenBytes))
	if err != nil {
		s.logger.Warn().Err(err).Msg("refresh token expired, clearing session")
		s.repo.DeleteUserSessions(ctx, userID)
		return &AuthState{Authenticated: false}, nil
	}

	// Get user from claims
	claims, _ := s.jwt.ValidateToken(tokenPair.AccessToken)
	var user *User
	if claims != nil {
		user = &User{
			ID:       claims.UserID,
			GitHubID: claims.GitHubID,
			Username: claims.Username,
		}
	}

	s.logger.Info().
		Str("user_id", userID).
		Msg("session restored successfully")

	return &AuthState{
		Authenticated: true,
		User:          user,
		AccessToken:   tokenPair.AccessToken,
		ExpiresAt:     tokenPair.ExpiresAt,
	}, nil
}

// Logout removes all sessions for the current user.
func (s *Service) Logout(ctx context.Context, userID string) error {
	s.logger.Info().Str("user_id", userID).Msg("logging out")
	return s.repo.DeleteUserSessions(ctx, userID)
}
