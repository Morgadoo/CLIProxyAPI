package redpill

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
	log "github.com/sirupsen/logrus"
)

const (
	// RedPill chat API endpoints
	RedPillChatCompletionsEndpoint = "https://chat.redpill.ai/api/chat/completions"
	RedPillModelsEndpoint          = "https://chat.redpill.ai/api/models"
)

// DefaultAPIBaseURL is the base URL for RedPill chat API.
const DefaultAPIBaseURL = "https://chat.redpill.ai/api"

// RedPillAuth encapsulates the HTTP client helpers for the cookie-based auth flow.
type RedPillAuth struct {
	httpClient *http.Client
}

// NewRedPillAuth constructs a new RedPillAuth with proxy-aware transport.
func NewRedPillAuth(cfg *config.Config) *RedPillAuth {
	client := &http.Client{Timeout: 30 * time.Second}
	return &RedPillAuth{httpClient: util.SetProxy(&cfg.SDKConfig, client)}
}

// RedPillTokenData captures processed token details from cookie authentication.
type RedPillTokenData struct {
	Email  string
	Expire string
	Cookie string
}

// jwtPayload represents the decoded JWT payload from RedPill.
type jwtPayload struct {
	UserID       int    `json:"user_id"`
	Email        string `json:"email"`
	JTI          string `json:"jti"`
	IsSubscribed bool   `json:"is_subscribed"`
	Exp          int64  `json:"exp"`
}

// AuthenticateWithCookie performs authentication using a JWT cookie from the browser.
func (ra *RedPillAuth) AuthenticateWithCookie(ctx context.Context, cookie string) (*RedPillTokenData, error) {
	if strings.TrimSpace(cookie) == "" {
		return nil, fmt.Errorf("redpill cookie authentication: cookie is empty")
	}

	// Extract the JWT token from the cookie
	token := ExtractToken(cookie)
	if token == "" {
		return nil, fmt.Errorf("redpill cookie authentication: no token found in cookie")
	}

	// Decode JWT to extract user info and expiration
	payload, err := decodeJWT(token)
	if err != nil {
		return nil, fmt.Errorf("redpill cookie authentication: %w", err)
	}

	if payload.Email == "" {
		return nil, fmt.Errorf("redpill cookie authentication: missing email in token")
	}

	// Check expiration
	if time.Now().Unix() >= payload.Exp {
		return nil, fmt.Errorf("redpill cookie authentication: token has expired")
	}

	// Verify the token works by making a test request to the models endpoint
	if err := ra.verifyToken(ctx, token); err != nil {
		return nil, fmt.Errorf("redpill cookie authentication: verification failed: %w", err)
	}

	expireTime := time.Unix(payload.Exp, 0).Format(time.RFC3339)

	data := &RedPillTokenData{
		Email:  payload.Email,
		Expire: expireTime,
		Cookie: cookie,
	}

	return data, nil
}

// verifyToken checks if the token is valid by making a simple API call.
func (ra *RedPillAuth) verifyToken(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, RedPillModelsEndpoint, nil)
	if err != nil {
		return fmt.Errorf("create verification request failed: %w", err)
	}

	req.Header.Set("Cookie", "token="+token)
	req.Header.Set("Accept", "application/json")

	resp, err := ra.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("verification request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("verification failed with status %d", resp.StatusCode)
	}

	return nil
}

// CreateCookieTokenStorage converts cookie-based token data into persistence storage.
func (ra *RedPillAuth) CreateCookieTokenStorage(data *RedPillTokenData) *RedPillTokenStorage {
	if data == nil {
		return nil
	}

	// Save just the token= part of the cookie
	token := ExtractToken(data.Cookie)
	cookieToSave := ""
	if token != "" {
		cookieToSave = "token=" + token + ";"
	}

	return &RedPillTokenStorage{
		Email:       data.Email,
		Cookie:      cookieToSave,
		Expire:      data.Expire,
		LastRefresh: time.Now().Format(time.RFC3339),
		Type:        "redpill",
	}
}

// ShouldRefreshToken checks if the JWT token is approaching expiration (within 2 days).
func ShouldRefreshToken(expireTime string) (bool, time.Duration, error) {
	if strings.TrimSpace(expireTime) == "" {
		return false, 0, fmt.Errorf("redpill: expire time is empty")
	}

	expire, err := time.Parse(time.RFC3339, expireTime)
	if err != nil {
		return false, 0, fmt.Errorf("redpill: parse expire time failed: %w", err)
	}

	now := time.Now()
	twoDaysFromNow := now.Add(48 * time.Hour)

	needsRefresh := expire.Before(twoDaysFromNow)
	timeUntilExpiry := expire.Sub(now)

	return needsRefresh, timeUntilExpiry, nil
}

// decodeJWT decodes a JWT payload without signature verification.
func decodeJWT(token string) (*jwtPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	payloadB64 := parts[1]
	// Add padding
	if m := len(payloadB64) % 4; m != 0 {
		payloadB64 += strings.Repeat("=", 4-m)
	}

	decoded, err := base64.URLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, fmt.Errorf("decode JWT payload failed: %w", err)
	}

	var payload jwtPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal JWT payload failed: %w", err)
	}

	log.Debugf("redpill JWT decoded: user_id=%d email=%s subscribed=%v", payload.UserID, payload.Email, payload.IsSubscribed)

	return &payload, nil
}
