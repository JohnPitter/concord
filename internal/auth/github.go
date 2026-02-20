package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog"
)

const (
	githubDeviceCodeURL = "https://github.com/login/device/code"
	githubAccessTokenURL = "https://github.com/login/oauth/access_token"
	githubUserAPIURL     = "https://api.github.com/user"
)

// DeviceCodeResponse is returned when requesting a device code from GitHub.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// GitHubUser represents the authenticated user's GitHub profile.
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubOAuth handles the GitHub Device Flow (RFC 8628).
type GitHubOAuth struct {
	clientID   string
	httpClient *http.Client
	logger     zerolog.Logger
}

// NewGitHubOAuth creates a new GitHub OAuth handler.
func NewGitHubOAuth(clientID string, logger zerolog.Logger) *GitHubOAuth {
	return &GitHubOAuth{
		clientID: clientID,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger.With().Str("component", "github_oauth").Logger(),
	}
}

// RequestDeviceCode initiates the device flow by requesting a device code.
// Complexity: O(1) — single HTTP request
func (g *GitHubOAuth) RequestDeviceCode(ctx context.Context) (*DeviceCodeResponse, error) {
	g.logger.Info().Msg("requesting device code from GitHub")

	data := url.Values{
		"client_id": {g.clientID},
		"scope":     {"read:user"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubDeviceCodeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.URL.RawQuery = data.Encode()
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request device code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub returned status %d: %s", resp.StatusCode, string(body))
	}

	var result DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	g.logger.Info().
		Str("user_code", result.UserCode).
		Str("verification_uri", result.VerificationURI).
		Int("expires_in", result.ExpiresIn).
		Msg("device code received")

	return &result, nil
}

// PollForToken polls GitHub until the user authorizes or the code expires.
// Complexity: O(n) where n = expires_in / interval
func (g *GitHubOAuth) PollForToken(ctx context.Context, deviceCode string, interval int) (string, error) {
	g.logger.Info().Msg("polling for access token")

	if interval < 5 {
		interval = 5
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			token, err := g.exchangeDeviceCode(ctx, deviceCode)
			if err == nil {
				g.logger.Info().Msg("access token received")
				return token, nil
			}

			if err == errAuthPending {
				continue
			}
			if err == errSlowDown {
				ticker.Reset(time.Duration(interval+5) * time.Second)
				continue
			}

			return "", err
		}
	}
}

var (
	errAuthPending = fmt.Errorf("authorization_pending")
	errSlowDown    = fmt.Errorf("slow_down")
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

func (g *GitHubOAuth) exchangeDeviceCode(ctx context.Context, deviceCode string) (string, error) {
	data := url.Values{
		"client_id":   {g.clientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubAccessTokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.URL.RawQuery = data.Encode()
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to exchange device code: %w", err)
	}
	defer resp.Body.Close()

	var result tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	switch result.Error {
	case "":
		return result.AccessToken, nil
	case "authorization_pending":
		return "", errAuthPending
	case "slow_down":
		return "", errSlowDown
	case "expired_token":
		return "", fmt.Errorf("device code expired — please restart login")
	case "access_denied":
		return "", fmt.Errorf("user denied authorization")
	default:
		return "", fmt.Errorf("GitHub OAuth error: %s", result.Error)
	}
}

// FetchUser retrieves the authenticated user's GitHub profile.
// Complexity: O(1) — single HTTP request
func (g *GitHubOAuth) FetchUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	g.logger.Info().Msg("fetching GitHub user profile")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubUserAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}

	g.logger.Info().
		Int64("github_id", user.ID).
		Str("login", user.Login).
		Msg("user profile fetched")

	return &user, nil
}
