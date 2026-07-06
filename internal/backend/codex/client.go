package codex

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/developerAkX/pigment/internal/auth"
)

// GenerateRequest holds all parameters for a generation request.
type GenerateRequest struct {
	Tokens       *auth.Tokens
	Payload      *RequestPayload
	TotalTimeout time.Duration
	StallTimeout time.Duration
	OnPhase      PhaseCallback
}

// GenerateResponse holds the result of a generation.
type GenerateResponse struct {
	ImageBytes    []byte
	RevisedPrompt string
	ItemMeta      map[string]interface{}
}

// Generate performs the image generation request against the codex backend.
func Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	codexVersion := auth.DetectCodexVersion()

	client, transport := newHTTPClient(req.TotalTimeout)
	defer transport.CloseIdleConnections()

	resp, err := doRequest(ctx, client, req, codexVersion)
	if err != nil {
		return nil, err
	}

	// Handle 401/403 with token refresh
	if (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) &&
		req.Tokens.RefreshToken != "" {
		resp.Body.Close()

		refreshResult, err := auth.RefreshAccessToken(
			req.Tokens.RefreshToken,
			codexVersion,
			auth.FormatOSInfo(),
		)
		if err != nil {
			return nil, err
		}

		// Update in-memory tokens
		req.Tokens.AccessToken = refreshResult.AccessToken
		if refreshResult.RefreshToken != "" {
			req.Tokens.RefreshToken = refreshResult.RefreshToken
		}

		// Persist — warning only on failure
		if err := auth.PersistRefreshedTokens(refreshResult); err != nil {
			fmt.Fprintf(getWarnWriter(), "warning: could not persist refreshed token to ~/.codex/auth.json: %v\n", err)
		}

		// Retry with new token
		resp, err = doRequest(ctx, client, req, codexVersion)
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()

	// Handle non-2xx
	if resp.StatusCode != http.StatusOK {
		if (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) &&
			req.Tokens.RefreshToken == "" {
			bodyPrefix := readBodyPrefix(resp.Body, 600)
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, bodyPrefix)
		}
		bodyPrefix := readBodyPrefix(resp.Body, 600)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, bodyPrefix)
	}

	// Parse SSE stream with stall timeout
	stallReader := newStallTimeoutReader(ctx, resp.Body, resp.Body, req.StallTimeout, req.TotalTimeout)

	// Track the last phase on the stall reader so timeout errors can
	// report where the stream got stuck.
	onPhase := func(phase string, partialCount int) {
		stallReader.setPhase(phase)
		if req.OnPhase != nil {
			req.OnPhase(phase, partialCount)
		}
	}
	streamResult := ParseSSEStream(stallReader, onPhase)

	if readErr, phase, totalExpired := stallReader.state(); readErr != nil {
		elapsed := time.Since(stallReader.startTime).Seconds()
		if phase == "" {
			phase = "none"
		}
		if totalExpired {
			return nil, fmt.Errorf(
				"timed out: no image within the %.0fs total budget (last phase: %s, %.1fs elapsed). Raise --timeout for very large images.",
				req.TotalTimeout.Seconds(), phase, elapsed,
			)
		}
		return nil, fmt.Errorf(
			"stalled: the image backend sent no data for ~%.0fs (last phase: %s, %.1fs elapsed). It may be overloaded — retry, or raise --stall-timeout / --timeout.",
			req.StallTimeout.Seconds(), phase, elapsed,
		)
	}

	if streamResult.Image == nil {
		return nil, fmt.Errorf("%s", streamResult.Error)
	}

	// Decode base64
	imgBytes, err := base64.StdEncoding.DecodeString(streamResult.Image.Base64Data)
	if err != nil {
		return nil, fmt.Errorf("backend returned invalid base64 in image_generation_call.result")
	}

	return &GenerateResponse{
		ImageBytes:    imgBytes,
		RevisedPrompt: streamResult.Image.RevisedPrompt,
		ItemMeta:      streamResult.Image.ItemMeta,
	}, nil
}

// newHTTPClient builds a client shared across the initial request and the
// post-refresh retry, so the retry reuses the same connection pool.
func newHTTPClient(totalTimeout time.Duration) (*http.Client, *http.Transport) {
	connectTimeout := 30 * time.Second
	if totalTimeout < connectTimeout {
		connectTimeout = totalTimeout
	}
	if connectTimeout < time.Second {
		connectTimeout = time.Second
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: connectTimeout,
		}).DialContext,
		DisableCompression: true,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   0, // We manage timeout ourselves
	}, transport
}

func doRequest(ctx context.Context, client *http.Client, req *GenerateRequest, codexVersion string) (*http.Response, error) {
	payloadBytes, err := MarshalPayload(req.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", CodexEndpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("network error contacting the image backend: %v", err)
	}

	SetHeaders(httpReq, req.Tokens, codexVersion)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("network error contacting the image backend: %v", err)
	}
	return resp, nil
}

func readBodyPrefix(body io.Reader, maxBytes int) string {
	buf := make([]byte, maxBytes)
	n, _ := io.ReadAtLeast(body, buf, 1)
	return string(buf[:n])
}

// warnWriter is where warnings go. Nil until SetWarnWriter is called;
// getWarnWriter falls back to io.Discard so writes are always safe.
var warnWriter io.Writer

func getWarnWriter() io.Writer {
	if warnWriter != nil {
		return warnWriter
	}
	return io.Discard
}

// SetWarnWriter sets the writer for warning messages.
func SetWarnWriter(w io.Writer) {
	warnWriter = w
}
