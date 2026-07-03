// Package codex implements the HTTP client for the ChatGPT codex responses endpoint.
package codex

import "encoding/json"

const (
	CodexEndpoint = "https://chatgpt.com/backend-api/codex/responses"
	Instructions  = "You are an image generation assistant."
)

// RequestPayload is the top-level Responses API request body.
type RequestPayload struct {
	Model             string          `json:"model"`
	Stream            bool            `json:"stream"`
	Instructions      string          `json:"instructions"`
	Input             []InputMessage  `json:"input"`
	Tools             []ImageTool     `json:"tools"`
	ToolChoice        string          `json:"tool_choice"`
	ParallelToolCalls bool            `json:"parallel_tool_calls"`
	Store             bool            `json:"store"`
	Reasoning         ReasoningConfig `json:"reasoning"`
	Include           []string        `json:"include"`
	Text              TextConfig      `json:"text"`
}

// InputMessage is a single user message.
type InputMessage struct {
	Type    string        `json:"type"`
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

// ContentPart is either input_text or input_image.
type ContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// ImageTool is the image_generation tool config.
type ImageTool struct {
	Type         string `json:"type"`
	OutputFormat string `json:"output_format"`
	Size         string `json:"size,omitempty"` // omitted when "auto"
}

// ReasoningConfig holds reasoning parameters.
type ReasoningConfig struct {
	Effort  string `json:"effort"`
	Summary string `json:"summary"`
}

// TextConfig holds text parameters.
type TextConfig struct {
	Verbosity string `json:"verbosity"`
}

// PayloadOptions holds parameters for building a request payload.
type PayloadOptions struct {
	Model  string
	Prompt string // the composed prompt text (with preamble)
	Format string // png, jpeg, webp
	Size   string // "auto" means omit

	// RefDataURIs are data URIs for reference images, in order.
	RefDataURIs []string
	HasRefs     bool // true if any references are present
}

// BuildPayload constructs a RequestPayload from the given options.
func BuildPayload(opts PayloadOptions) *RequestPayload {
	var content []ContentPart

	// Reference images come first
	for _, uri := range opts.RefDataURIs {
		content = append(content, ContentPart{
			Type:     "input_image",
			ImageURL: uri,
		})
	}

	// Then the text prompt
	content = append(content, ContentPart{
		Type: "input_text",
		Text: opts.Prompt,
	})

	toolChoice := "auto"
	if opts.HasRefs {
		toolChoice = "required"
	}

	tool := ImageTool{
		Type:         "image_generation",
		OutputFormat: opts.Format,
	}
	if opts.Size != "auto" && opts.Size != "" {
		tool.Size = opts.Size
	}

	return &RequestPayload{
		Model:        opts.Model,
		Stream:       true,
		Instructions: Instructions,
		Input: []InputMessage{
			{
				Type:    "message",
				Role:    "user",
				Content: content,
			},
		},
		Tools:             []ImageTool{tool},
		ToolChoice:        toolChoice,
		ParallelToolCalls: false,
		Store:             false,
		Reasoning: ReasoningConfig{
			Effort:  "low",
			Summary: "auto",
		},
		Include: []string{"reasoning.encrypted_content"},
		Text:    TextConfig{Verbosity: "low"},
	}
}

// MarshalPayload serializes a RequestPayload to JSON bytes.
func MarshalPayload(p *RequestPayload) ([]byte, error) {
	return json.Marshal(p)
}
