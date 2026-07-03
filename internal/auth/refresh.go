package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	OAuthTokenURL = "https://auth.openai.com/oauth/token"
	OAuthClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
)

// RefreshResult holds the new tokens from a successful refresh.
type RefreshResult struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
}

// RefreshAccessToken performs a token refresh using the given refresh token.
// versionStr is the codex version string for the User-Agent header.
func RefreshAccessToken(refreshToken, versionStr, osInfo string) (*RefreshResult, error) {
	form := url.Values{
		"client_id":     {OAuthClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {"openid profile email"},
	}

	req, err := http.NewRequest("POST", OAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %v", err)
	}

	ua := fmt.Sprintf("codex_cli_rs/%s (%s) pigment", versionStr, osInfo)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", ua)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Error string `json:"error"`
		}
		json.Unmarshal(body, &errBody)

		if errBody.Error == "invalid_grant" {
			return nil, fmt.Errorf(
				"refresh_token is no longer valid — run `codex login` again to re-authenticate.",
			)
		}
		msg := fmt.Sprintf("token refresh failed: HTTP %d", resp.StatusCode)
		if errBody.Error != "" {
			msg += fmt.Sprintf(" (%s)", errBody.Error)
		}
		return nil, fmt.Errorf("%s", msg)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("token refresh failed: invalid response JSON")
	}

	r := &RefreshResult{}
	if v, ok := result["access_token"].(string); ok {
		r.AccessToken = v
	}
	if v, ok := result["refresh_token"].(string); ok {
		r.RefreshToken = v
	}
	if v, ok := result["id_token"].(string); ok {
		r.IDToken = v
	}

	if r.AccessToken == "" {
		// Build list of present string fields
		var fields []string
		for k, v := range result {
			if _, ok := v.(string); ok {
				fields = append(fields, k)
			}
		}
		return nil, fmt.Errorf(
			"refresh response missing access_token (present string fields: %v)", fields,
		)
	}

	return r, nil
}

// PersistRefreshedTokens atomically updates auth.json with new tokens.
func PersistRefreshedTokens(result *RefreshResult) error {
	authPath := AuthFilePath()

	// Read existing file
	data, err := os.ReadFile(authPath)
	if err != nil {
		return err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	tokens, _ := raw["tokens"].(map[string]interface{})
	if tokens == nil {
		tokens = make(map[string]interface{})
	}

	if result.AccessToken != "" {
		tokens["access_token"] = result.AccessToken
	}
	if result.RefreshToken != "" {
		tokens["refresh_token"] = result.RefreshToken
	}
	if result.IDToken != "" {
		tokens["id_token"] = result.IDToken
	}
	raw["tokens"] = tokens
	raw["last_refresh"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")

	newData, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file in same directory
	dir := filepath.Dir(authPath)
	tmpFile, err := os.CreateTemp(dir, ".auth.*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	defer func() {
		if tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()

	if err := os.Chmod(tmpPath, 0600); err != nil {
		tmpFile.Close()
		return err
	}

	if _, err := tmpFile.Write(newData); err != nil {
		tmpFile.Close()
		return err
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, authPath); err != nil {
		return err
	}

	tmpPath = "" // prevent deferred cleanup
	return nil
}
