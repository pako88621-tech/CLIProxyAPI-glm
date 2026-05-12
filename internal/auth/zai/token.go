// Package zai provides authentication and token management functionality
// for Z.ai (ChatGLM) services.
package zai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/misc"
)

// ZaiTokenStorage stores token information for Z.ai API authentication.
type ZaiTokenStorage struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Expired     string `json:"expired,omitempty"`
	Type        string `json:"type"`

	Metadata map[string]any `json:"-"`
}

func (ts *ZaiTokenStorage) SetMetadata(meta map[string]any) {
	ts.Metadata = meta
}

func (ts *ZaiTokenStorage) SaveTokenToFile(authFilePath string) error {
	misc.LogSavingCredentials(authFilePath)
	ts.Type = "zai"

	if err := os.MkdirAll(filepath.Dir(authFilePath), 0700); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	f, err := os.Create(authFilePath)
	if err != nil {
		return fmt.Errorf("failed to create token file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	data, errMerge := misc.MergeMetadata(ts, ts.Metadata)
	if errMerge != nil {
		return fmt.Errorf("failed to merge metadata: %w", errMerge)
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to write token to file: %w", err)
	}
	return nil
}

func (ts *ZaiTokenStorage) IsExpired() bool {
	if ts.Expired == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, ts.Expired)
	if err != nil {
		return true
	}
	return time.Now().Add(5 * time.Minute).After(t)
}

func (ts *ZaiTokenStorage) NeedsRefresh() bool {
	// For Z.ai Guest tokens, we just get a new one when it expires.
	// We don't have a refresh token flow yet for Guest.
	return ts.IsExpired()
}
