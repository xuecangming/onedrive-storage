package onedrive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TokenResponse represents OAuth2 token response
type TokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// AuthConfig represents OAuth2 configuration
type AuthConfig struct {
	ClientID     string
	ClientSecret string
	TenantID     string
	RedirectURI  string
}

// Auth handles OneDrive OAuth2 authentication
type Auth struct {
	config     AuthConfig
	httpClient *http.Client
}

// NewAuth creates a new Auth instance
func NewAuth(config AuthConfig) *Auth {
	return &Auth{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAuthorizationURL returns the URL for user authorization
func (a *Auth) GetAuthorizationURL(state string) string {
	params := url.Values{}
	params.Add("client_id", a.config.ClientID)
	params.Add("response_type", "code")
	params.Add("redirect_uri", a.config.RedirectURI)
	params.Add("response_mode", "query")
	params.Add("scope", "offline_access Files.ReadWrite.All")
	params.Add("state", state)

	baseURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", a.config.TenantID)
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

// ExchangeCode exchanges authorization code for access token
func (a *Auth) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", a.config.TenantID)

	data := url.Values{}
	data.Set("client_id", a.config.ClientID)
	data.Set("client_secret", a.config.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", a.config.RedirectURI)
	data.Set("grant_type", "authorization_code")
	data.Set("scope", "offline_access Files.ReadWrite.All")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokenResp, nil
}

// RefreshToken refreshes an access token using a refresh token
func (a *Auth) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", a.config.TenantID)

	data := url.Values{}
	data.Set("client_id", a.config.ClientID)
	data.Set("client_secret", a.config.ClientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")
	data.Set("scope", "offline_access Files.ReadWrite.All")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokenResp, nil
}

// ValidateToken checks if a token is valid
func (a *Auth) ValidateToken(ctx context.Context, accessToken string) (bool, error) {
	// Try to get drive info to validate token
	client := NewClient(accessToken)
	_, err := client.GetDrive(ctx)
	if err != nil {
		return false, nil
	}
	return true, nil
}
