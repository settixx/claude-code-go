package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OAuthConfig holds OAuth2 credentials and endpoints for an MCP server.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	AuthURL      string
	Scopes       []string
	RedirectURI  string
}

// OAuthToken represents an OAuth2 token with optional refresh capability.
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresIn    int64     `json:"expires_in,omitempty"`
	ExpiresAt    time.Time `json:"-"`
}

// Valid reports whether the token is present and not expired.
func (t *OAuthToken) Valid() bool {
	if t == nil || t.AccessToken == "" {
		return false
	}
	if t.ExpiresAt.IsZero() {
		return true
	}
	return time.Now().Before(t.ExpiresAt.Add(-30 * time.Second))
}

// TokenManager handles OAuth2 token lifecycle for MCP server authentication.
type TokenManager struct {
	config     OAuthConfig
	token      *OAuthToken
	mu         sync.Mutex
	httpClient *http.Client
}

// NewTokenManager creates a TokenManager with the given configuration.
func NewTokenManager(cfg OAuthConfig) *TokenManager {
	return &TokenManager{
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetToken returns a valid access token, refreshing if necessary.
func (tm *TokenManager) GetToken(ctx context.Context) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.token.Valid() {
		return tm.token.AccessToken, nil
	}

	if tm.token != nil && tm.token.RefreshToken != "" {
		if err := tm.refreshTokenLocked(ctx); err == nil {
			return tm.token.AccessToken, nil
		}
	}

	return "", fmt.Errorf("no valid token available; authorization required")
}

// RefreshToken forces a token refresh using the stored refresh token.
func (tm *TokenManager) RefreshToken(ctx context.Context) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.refreshTokenLocked(ctx)
}

func (tm *TokenManager) refreshTokenLocked(ctx context.Context) error {
	if tm.token == nil || tm.token.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	values := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {tm.token.RefreshToken},
		"client_id":     {tm.config.ClientID},
	}
	if tm.config.ClientSecret != "" {
		values.Set("client_secret", tm.config.ClientSecret)
	}

	return tm.doTokenRequest(ctx, values)
}

// HandleAuthRedirect exchanges an authorization code for tokens.
func (tm *TokenManager) HandleAuthRedirect(ctx context.Context, code string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	values := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"client_id":    {tm.config.ClientID},
		"redirect_uri": {tm.config.RedirectURI},
	}
	if tm.config.ClientSecret != "" {
		values.Set("client_secret", tm.config.ClientSecret)
	}

	return tm.doTokenRequest(ctx, values)
}

func (tm *TokenManager) doTokenRequest(ctx context.Context, values url.Values) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tm.config.TokenURL,
		strings.NewReader(values.Encode()))
	if err != nil {
		return fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := tm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return fmt.Errorf("decode token response: %w", err)
	}

	if token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	if token.RefreshToken == "" && tm.token != nil {
		token.RefreshToken = tm.token.RefreshToken
	}

	tm.token = &token
	return nil
}

// AuthorizationURL builds the URL the user should visit to authorize this client.
func (tm *TokenManager) AuthorizationURL(state string) string {
	v := url.Values{
		"client_id":     {tm.config.ClientID},
		"response_type": {"code"},
		"redirect_uri":  {tm.config.RedirectURI},
		"state":         {state},
	}
	if len(tm.config.Scopes) > 0 {
		v.Set("scope", strings.Join(tm.config.Scopes, " "))
	}
	return tm.config.AuthURL + "?" + v.Encode()
}

// SetToken sets the token directly (e.g. loaded from persistent storage).
func (tm *TokenManager) SetToken(token *OAuthToken) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.token = token
}

// Token returns the current token (may be nil or expired).
func (tm *TokenManager) Token() *OAuthToken {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.token
}
