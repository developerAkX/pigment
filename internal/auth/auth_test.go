package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTokensFrom_Valid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "auth.json")
	data := `{
		"tokens": {
			"access_token": "eyJtoken123",
			"account_id": "acct_abc123",
			"refresh_token": "v1.refresh456"
		},
		"last_refresh": "2026-07-01T12:00:00Z"
	}`
	os.WriteFile(p, []byte(data), 0600)

	tok, err := LoadTokensFrom(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "eyJtoken123" {
		t.Errorf("access_token = %q, want %q", tok.AccessToken, "eyJtoken123")
	}
	if tok.AccountID != "acct_abc123" {
		t.Errorf("account_id = %q, want %q", tok.AccountID, "acct_abc123")
	}
	if tok.RefreshToken != "v1.refresh456" {
		t.Errorf("refresh_token = %q, want %q", tok.RefreshToken, "v1.refresh456")
	}
	if tok.LastRefresh.IsZero() {
		t.Error("last_refresh should not be zero")
	}
}

func TestLoadTokensFrom_MissingFile(t *testing.T) {
	_, err := LoadTokensFrom("/nonexistent/auth.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	want := "~/.codex/auth.json not found."
	if got := err.Error(); len(got) < len(want) || got[:len(want)] != want {
		t.Errorf("error = %q, want prefix %q", got, want)
	}
}

func TestLoadTokensFrom_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "auth.json")
	os.WriteFile(p, []byte("not json{{{"), 0600)

	_, err := LoadTokensFrom(p)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	want := "failed to parse ~/.codex/auth.json:"
	if got := err.Error(); len(got) < len(want) || got[:len(want)] != want {
		t.Errorf("error = %q, want prefix %q", got, want)
	}
}

func TestLoadTokensFrom_NoAccessToken(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "auth.json")
	data := `{"tokens": {"account_id": "acct_123"}}`
	os.WriteFile(p, []byte(data), 0600)

	_, err := LoadTokensFrom(p)
	if err == nil {
		t.Fatal("expected error for missing access_token")
	}
	want := "no ChatGPT OAuth access_token"
	if got := err.Error(); len(got) < len(want) || got[:len(want)] != want {
		t.Errorf("error = %q, want prefix %q", got, want)
	}
}

func TestLoadTokensFrom_APIKeyOnly(t *testing.T) {
	// An auth.json with OPENAI_API_KEY but no access_token should be rejected
	dir := t.TempDir()
	p := filepath.Join(dir, "auth.json")
	data := `{"tokens": {"OPENAI_API_KEY": "sk-123456"}}`
	os.WriteFile(p, []byte(data), 0600)

	_, err := LoadTokensFrom(p)
	if err == nil {
		t.Fatal("expected error for API key only file")
	}
	want := "no ChatGPT OAuth access_token"
	if got := err.Error(); len(got) < len(want) || got[:len(want)] != want {
		t.Errorf("error = %q, want prefix %q", got, want)
	}
}

func TestLoadTokensFrom_NonStringAccessToken(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"null_token", `{"tokens": {"access_token": null}}`},
		{"number_token", `{"tokens": {"access_token": 12345}}`},
		{"bool_token", `{"tokens": {"access_token": true}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "auth.json")
			os.WriteFile(p, []byte(tc.json), 0600)

			_, err := LoadTokensFrom(p)
			if err == nil {
				t.Fatal("expected error for non-string access_token")
			}
		})
	}
}

func TestLoadTokensFrom_NoTokensObject(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "auth.json")
	os.WriteFile(p, []byte(`{}`), 0600)

	_, err := LoadTokensFrom(p)
	if err == nil {
		t.Fatal("expected error for missing tokens object")
	}
	want := "no ChatGPT OAuth access_token"
	if got := err.Error(); len(got) < len(want) || got[:len(want)] != want {
		t.Errorf("error = %q, want prefix %q", got, want)
	}
}

func TestLoadTokensFrom_OptionalFieldsMissing(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "auth.json")
	data := `{"tokens": {"access_token": "eyJtoken"}}`
	os.WriteFile(p, []byte(data), 0600)

	tok, err := LoadTokensFrom(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccountID != "" {
		t.Errorf("account_id should be empty, got %q", tok.AccountID)
	}
	if tok.RefreshToken != "" {
		t.Errorf("refresh_token should be empty, got %q", tok.RefreshToken)
	}
}
