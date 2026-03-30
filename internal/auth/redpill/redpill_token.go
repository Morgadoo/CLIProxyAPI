package redpill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/misc"
)

// RedPillTokenStorage persists RedPill cookie-based credentials.
type RedPillTokenStorage struct {
	Email       string `json:"email"`
	Cookie      string `json:"cookie"`
	Expire      string `json:"expired"`
	LastRefresh string `json:"last_refresh"`
	Type        string `json:"type"`

	// Metadata holds arbitrary key-value pairs injected via hooks.
	Metadata map[string]any `json:"-"`
}

// SetMetadata allows external callers to inject metadata into the storage before saving.
func (ts *RedPillTokenStorage) SetMetadata(meta map[string]any) {
	ts.Metadata = meta
}

// SaveTokenToFile serialises the token storage to disk.
func (ts *RedPillTokenStorage) SaveTokenToFile(authFilePath string) error {
	misc.LogSavingCredentials(authFilePath)
	ts.Type = "redpill"
	if err := os.MkdirAll(filepath.Dir(authFilePath), 0o700); err != nil {
		return fmt.Errorf("redpill token: create directory failed: %w", err)
	}

	f, err := os.Create(authFilePath)
	if err != nil {
		return fmt.Errorf("redpill token: create file failed: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Merge metadata using helper
	data, errMerge := misc.MergeMetadata(ts, ts.Metadata)
	if errMerge != nil {
		return fmt.Errorf("failed to merge metadata: %w", errMerge)
	}

	if err = json.NewEncoder(f).Encode(data); err != nil {
		return fmt.Errorf("redpill token: encode token failed: %w", err)
	}
	return nil
}
