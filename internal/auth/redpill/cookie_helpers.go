package redpill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NormalizeCookie normalizes raw cookie strings for RedPill authentication.
// For RedPill, the cookie must contain the "token" field (JWT).
func NormalizeCookie(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("cookie cannot be empty")
	}

	// If it looks like a raw JWT (starts with eyJ), wrap it as token=<jwt>
	if strings.HasPrefix(trimmed, "eyJ") && !strings.Contains(trimmed, "=") {
		trimmed = "token=" + trimmed
	}

	combined := strings.Join(strings.Fields(trimmed), " ")
	if !strings.HasSuffix(combined, ";") {
		combined += ";"
	}
	if !strings.Contains(combined, "token=") {
		return "", fmt.Errorf("cookie missing token field")
	}
	return combined, nil
}

// ExtractToken extracts the JWT token value from a cookie string.
func ExtractToken(cookie string) string {
	parts := strings.Split(cookie, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "token=") {
			return strings.TrimPrefix(part, "token=")
		}
	}
	return ""
}

// SanitizeRedPillFileName normalizes user identifiers for safe filename usage.
func SanitizeRedPillFileName(raw string) string {
	if raw == "" {
		return ""
	}
	var result strings.Builder
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '@' || r == '.' || r == '-' {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}

// CheckDuplicateToken checks if the given JWT token already exists in any redpill auth file.
// Returns the path of the existing file if found, empty string otherwise.
func CheckDuplicateToken(authDir, token string) (string, error) {
	if token == "" {
		return "", nil
	}

	entries, err := os.ReadDir(authDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read auth dir failed: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "redpill-") || !strings.HasSuffix(name, ".json") {
			continue
		}

		filePath := filepath.Join(authDir, name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var tokenData struct {
			Cookie string `json:"cookie"`
		}
		if err := json.Unmarshal(data, &tokenData); err != nil {
			continue
		}

		existingToken := ExtractToken(tokenData.Cookie)
		if existingToken != "" && existingToken == token {
			return filePath, nil
		}
	}

	return "", nil
}
