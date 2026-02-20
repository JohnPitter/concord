package auth

import (
	"testing"
	"time"
)

const testSecret = "test-secret-must-be-at-least-32-characters-long"

func TestNewJWTManager_ValidSecret(t *testing.T) {
	mgr, err := NewJWTManager(testSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestNewJWTManager_ShortSecret(t *testing.T) {
	_, err := NewJWTManager("short")
	if err == nil {
		t.Fatal("expected error for short secret")
	}
}

func TestGenerateTokenPair(t *testing.T) {
	mgr, err := NewJWTManager(testSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pair, err := mgr.GenerateTokenPair("gh_12345", 12345, "testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pair.AccessToken == "" {
		t.Error("access token should not be empty")
	}
	if pair.RefreshToken == "" {
		t.Error("refresh token should not be empty")
	}
	if pair.AccessToken == pair.RefreshToken {
		t.Error("access and refresh tokens should be different")
	}
	if pair.ExpiresAt <= time.Now().Unix() {
		t.Error("expires_at should be in the future")
	}
}

func TestValidateToken_Valid(t *testing.T) {
	mgr, err := NewJWTManager(testSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pair, err := mgr.GenerateTokenPair("gh_12345", 12345, "testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	claims, err := mgr.ValidateToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if claims.UserID != "gh_12345" {
		t.Errorf("expected user_id gh_12345, got %s", claims.UserID)
	}
	if claims.GitHubID != 12345 {
		t.Errorf("expected github_id 12345, got %d", claims.GitHubID)
	}
	if claims.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", claims.Username)
	}
	if claims.Issuer != "concord" {
		t.Errorf("expected issuer concord, got %s", claims.Issuer)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	mgr, err := NewJWTManager(testSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = mgr.ValidateToken("invalid.token.here")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	mgr1, _ := NewJWTManager(testSecret)
	mgr2, _ := NewJWTManager("another-secret-must-be-at-least-32-characters-long")

	pair, _ := mgr1.GenerateTokenPair("gh_12345", 12345, "testuser")

	_, err := mgr2.ValidateToken(pair.AccessToken)
	if err == nil {
		t.Fatal("expected error when validating with wrong secret")
	}
}

func TestRefreshAccessToken(t *testing.T) {
	mgr, err := NewJWTManager(testSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	original, err := mgr.GenerateTokenPair("gh_99", 99, "refreshuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newPair, err := mgr.RefreshAccessToken(original.RefreshToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if newPair.AccessToken == "" {
		t.Error("new access token should not be empty")
	}

	claims, err := mgr.ValidateToken(newPair.AccessToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != "gh_99" {
		t.Errorf("expected user_id gh_99, got %s", claims.UserID)
	}
}

func TestRefreshAccessToken_RejectsAccessToken(t *testing.T) {
	mgr, _ := NewJWTManager(testSecret)

	pair, _ := mgr.GenerateTokenPair("gh_12345", 12345, "testuser")

	// Access token has issuer "concord", not "concord-refresh"
	_, err := mgr.RefreshAccessToken(pair.AccessToken)
	if err == nil {
		t.Fatal("expected error when using access token as refresh token")
	}
}
