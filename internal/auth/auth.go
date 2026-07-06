// Package auth handles reading and refreshing OAuth tokens from ~/.codex/auth.json.
package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Tokens holds the parsed authentication tokens.
type Tokens struct {
	AccessToken  string
	AccountID    string // may be empty
	RefreshToken string // may be empty
	LastRefresh  time.Time
}

// codexPath returns the path to a file inside ~/.codex, or an error if
// the home directory cannot be determined (e.g. $HOME unset in a container).
func codexPath(file string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory (is $HOME set?): %w", err)
	}
	return filepath.Join(home, ".codex", file), nil
}

// AuthFilePath returns the fixed path to ~/.codex/auth.json.
// For display purposes; callers that read/write should use codexPath
// to surface home-directory resolution errors.
func AuthFilePath() string {
	p, _ := codexPath("auth.json")
	return p
}

// VersionFilePath returns the fixed path to ~/.codex/version.json.
func VersionFilePath() string {
	p, _ := codexPath("version.json")
	return p
}

// rawAuthFile is the JSON structure of auth.json.
type rawAuthFile struct {
	Tokens      map[string]interface{} `json:"tokens"`
	LastRefresh string                 `json:"last_refresh"`
}

// LoadTokens reads and parses ~/.codex/auth.json.
func LoadTokens() (*Tokens, error) {
	path, err := codexPath("auth.json")
	if err != nil {
		return nil, err
	}
	return LoadTokensFrom(path)
}

// LoadTokensFrom reads and parses auth.json from the given path.
func LoadTokensFrom(path string) (*Tokens, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(
				"~/.codex/auth.json not found.\n" +
					"Install Codex CLI (`npm i -g @openai/codex`) and run `codex login` first.",
			)
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var raw rawAuthFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse ~/.codex/auth.json: %v", err)
	}

	if raw.Tokens == nil {
		return nil, fmt.Errorf(
			"no ChatGPT OAuth access_token in ~/.codex/auth.json. " +
				"Run `codex login` to sign into your ChatGPT subscription. " +
				"(An OPENAI_API_KEY in this file is not a substitute — " +
				"the backend requires a subscription OAuth token.)",
		)
	}

	t := &Tokens{}

	// Extract access_token (must be a string)
	if v, ok := raw.Tokens["access_token"]; ok {
		if s, ok := v.(string); ok {
			t.AccessToken = s
		}
	}

	if t.AccessToken == "" {
		return nil, fmt.Errorf(
			"no ChatGPT OAuth access_token in ~/.codex/auth.json. " +
				"Run `codex login` to sign into your ChatGPT subscription. " +
				"(An OPENAI_API_KEY in this file is not a substitute — " +
				"the backend requires a subscription OAuth token.)",
		)
	}

	// Extract optional fields
	if v, ok := raw.Tokens["account_id"]; ok {
		if s, ok := v.(string); ok {
			t.AccountID = s
		}
	}
	if v, ok := raw.Tokens["refresh_token"]; ok {
		if s, ok := v.(string); ok {
			t.RefreshToken = s
		}
	}

	// Parse last_refresh
	if raw.LastRefresh != "" {
		if lr, err := time.Parse("2006-01-02T15:04:05Z", raw.LastRefresh); err == nil {
			t.LastRefresh = lr
		}
	}

	return t, nil
}

// TokenPresent returns true if an access token can be loaded.
func TokenPresent() bool {
	t, err := LoadTokens()
	return err == nil && t.AccessToken != ""
}
