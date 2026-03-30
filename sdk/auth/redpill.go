package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/redpill"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
)

// RedPillAuthenticator implements the cookie-based login flow for RedPill accounts.
type RedPillAuthenticator struct{}

// NewRedPillAuthenticator constructs a new authenticator instance.
func NewRedPillAuthenticator() *RedPillAuthenticator { return &RedPillAuthenticator{} }

// Provider returns the provider key for the authenticator.
func (a *RedPillAuthenticator) Provider() string { return "redpill" }

// RefreshLead indicates how soon before expiry a refresh should be attempted.
// RedPill JWT tokens are long-lived (~30 days) and cannot be refreshed programmatically.
// We check 48 hours before expiry to warn the user.
func (a *RedPillAuthenticator) RefreshLead() *time.Duration {
	d := 48 * time.Hour
	return &d
}

// Login performs the cookie-based login flow. The user must provide a JWT cookie
// from their browser session at chat.redpill.ai.
func (a *RedPillAuthenticator) Login(ctx context.Context, cfg *config.Config, opts *LoginOptions) (*coreauth.Auth, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cliproxy auth: configuration is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if opts == nil {
		opts = &LoginOptions{}
	}

	promptFn := opts.Prompt
	if promptFn == nil {
		return nil, fmt.Errorf("redpill auth: prompt function is required for cookie login")
	}

	// Prompt user for the cookie
	cookieInput, err := promptFn("Enter RedPill cookie (JWT token from chat.redpill.ai): ")
	if err != nil {
		return nil, fmt.Errorf("redpill auth: failed to read cookie: %w", err)
	}

	cookie, err := redpill.NormalizeCookie(cookieInput)
	if err != nil {
		return nil, fmt.Errorf("redpill auth: %w", err)
	}

	// Check for duplicates
	token := redpill.ExtractToken(cookie)
	if existingFile, errDup := redpill.CheckDuplicateToken(cfg.AuthDir, token); errDup != nil {
		return nil, fmt.Errorf("redpill auth: failed to check duplicates: %w", errDup)
	} else if existingFile != "" {
		return nil, fmt.Errorf("redpill auth: duplicate token found in %s", existingFile)
	}

	// Authenticate
	authSvc := redpill.NewRedPillAuth(cfg)
	tokenData, err := authSvc.AuthenticateWithCookie(ctx, cookie)
	if err != nil {
		return nil, fmt.Errorf("redpill authentication failed: %w", err)
	}

	tokenStorage := authSvc.CreateCookieTokenStorage(tokenData)

	email := strings.TrimSpace(tokenStorage.Email)
	if email == "" {
		return nil, fmt.Errorf("redpill authentication failed: missing account identifier")
	}

	fileName := fmt.Sprintf("redpill-%s-%d.json", redpill.SanitizeRedPillFileName(email), time.Now().Unix())
	metadata := map[string]any{
		"email":        email,
		"expired":      tokenStorage.Expire,
		"cookie":       tokenStorage.Cookie,
		"type":         tokenStorage.Type,
		"last_refresh": tokenStorage.LastRefresh,
	}

	fmt.Println("RedPill authentication successful")

	return &coreauth.Auth{
		ID:       fileName,
		Provider: a.Provider(),
		FileName: fileName,
		Storage:  tokenStorage,
		Metadata: metadata,
	}, nil
}
