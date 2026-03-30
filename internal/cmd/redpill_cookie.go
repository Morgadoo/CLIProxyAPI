package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/redpill"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

// DoRedPillCookieAuth performs the RedPill cookie-based authentication.
func DoRedPillCookieAuth(cfg *config.Config, options *LoginOptions) {
	if options == nil {
		options = &LoginOptions{}
	}

	promptFn := options.Prompt
	if promptFn == nil {
		reader := bufio.NewReader(os.Stdin)
		promptFn = func(prompt string) (string, error) {
			fmt.Print(prompt)
			value, err := reader.ReadString('\n')
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(value), nil
		}
	}

	fmt.Println("RedPill Cookie Login")
	fmt.Println("====================")
	fmt.Println()
	fmt.Println("To get your JWT token:")
	fmt.Println("  1. Open https://chat.redpill.ai in your browser")
	fmt.Println("  2. Log in with your Pro account")
	fmt.Println("  3. Open DevTools (F12) > Application > Cookies")
	fmt.Println("  4. Copy the value of the 'token' cookie")
	fmt.Println()

	// Prompt user for cookie
	cookie, err := promptForRedPillCookie(promptFn)
	if err != nil {
		fmt.Printf("Failed to get cookie: %v\n", err)
		return
	}

	// Check for duplicate token before authentication
	token := redpill.ExtractToken(cookie)
	if existingFile, err := redpill.CheckDuplicateToken(cfg.AuthDir, token); err != nil {
		fmt.Printf("Failed to check duplicate: %v\n", err)
		return
	} else if existingFile != "" {
		fmt.Printf("Duplicate token found, authentication already exists: %s\n", filepath.Base(existingFile))
		return
	}

	// Authenticate with cookie
	auth := redpill.NewRedPillAuth(cfg)
	ctx := context.Background()

	tokenData, err := auth.AuthenticateWithCookie(ctx, cookie)
	if err != nil {
		fmt.Printf("RedPill cookie authentication failed: %v\n", err)
		return
	}

	// Create token storage
	tokenStorage := auth.CreateCookieTokenStorage(tokenData)

	// Get auth file path using email in filename
	authFilePath := getRedPillAuthFilePath(cfg, tokenData.Email)

	// Save token to file
	if err := tokenStorage.SaveTokenToFile(authFilePath); err != nil {
		fmt.Printf("Failed to save authentication: %v\n", err)
		return
	}

	fmt.Printf("Authentication successful! Email: %s\n", tokenData.Email)
	fmt.Printf("Expires at: %s\n", tokenData.Expire)
	fmt.Printf("Authentication saved to: %s\n", authFilePath)
}

// promptForRedPillCookie prompts the user to enter their RedPill cookie
func promptForRedPillCookie(promptFn func(string) (string, error)) (string, error) {
	line, err := promptFn("Enter RedPill Cookie (JWT token from browser): ")
	if err != nil {
		return "", fmt.Errorf("failed to read cookie: %w", err)
	}

	cookie, err := redpill.NormalizeCookie(line)
	if err != nil {
		return "", err
	}

	return cookie, nil
}

// getRedPillAuthFilePath returns the auth file path for a RedPill user
func getRedPillAuthFilePath(cfg *config.Config, email string) string {
	fileName := redpill.SanitizeRedPillFileName(email)
	if fileName == "" {
		fileName = fmt.Sprintf("redpill-%d", time.Now().UnixMilli())
	} else {
		fileName = fmt.Sprintf("redpill-%s", fileName)
	}
	return fmt.Sprintf("%s/%s-%d.json", cfg.AuthDir, fileName, time.Now().Unix())
}
