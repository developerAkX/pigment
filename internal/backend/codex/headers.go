package codex

import (
	"net/http"

	"github.com/developerAkX/pigment/internal/auth"
	"github.com/google/uuid"
)

// SetHeaders sets all required headers on an HTTP request per the spec.
func SetHeaders(req *http.Request, tokens *auth.Tokens, codexVersion string) {
	sessionID := uuid.New().String()

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("version", codexVersion)
	req.Header.Set("session_id", sessionID)
	req.Header.Set("x-client-request-id", sessionID)
	req.Header.Set("User-Agent", auth.FormatUserAgent(codexVersion))
	req.Header.Set("originator", "codex_cli_rs")

	if tokens.AccountID != "" {
		req.Header.Set("chatgpt-account-id", tokens.AccountID)
	}
}
