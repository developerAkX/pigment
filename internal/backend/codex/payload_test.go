package codex

import (
	"encoding/json"
	"testing"
)

func TestBuildPayload_NoRefs(t *testing.T) {
	p := BuildPayload(PayloadOptions{
		Model:   "gpt-5.5",
		Prompt:  "a watercolor cat",
		Format:  "png",
		Size:    "auto",
		HasRefs: false,
	})

	if p.Model != "gpt-5.5" {
		t.Errorf("model = %q, want %q", p.Model, "gpt-5.5")
	}
	if !p.Stream {
		t.Error("stream should be true")
	}
	if p.Instructions != Instructions {
		t.Errorf("instructions = %q, want %q", p.Instructions, Instructions)
	}
	if p.ToolChoice != "auto" {
		t.Errorf("tool_choice = %q, want %q", p.ToolChoice, "auto")
	}
	if p.ParallelToolCalls {
		t.Error("parallel_tool_calls should be false")
	}
	if p.Store {
		t.Error("store should be false")
	}

	// Check content has exactly 1 input_text part
	if len(p.Input) != 1 {
		t.Fatalf("input length = %d, want 1", len(p.Input))
	}
	content := p.Input[0].Content
	if len(content) != 1 {
		t.Fatalf("content length = %d, want 1", len(content))
	}
	if content[0].Type != "input_text" {
		t.Errorf("content[0].type = %q, want %q", content[0].Type, "input_text")
	}

	// Check tool — size should be omitted (empty string) since "auto"
	if len(p.Tools) != 1 {
		t.Fatalf("tools length = %d, want 1", len(p.Tools))
	}
	if p.Tools[0].Size != "" {
		t.Errorf("tool size = %q, want empty (auto)", p.Tools[0].Size)
	}
	if p.Tools[0].OutputFormat != "png" {
		t.Errorf("tool output_format = %q, want %q", p.Tools[0].OutputFormat, "png")
	}
}

func TestBuildPayload_WithRefs(t *testing.T) {
	p := BuildPayload(PayloadOptions{
		Model:       "gpt-5.5",
		Prompt:      "make it blue",
		Format:      "jpeg",
		Size:        "1024x1024",
		RefDataURIs: []string{"data:image/png;base64,abc123", "data:image/jpeg;base64,def456"},
		HasRefs:     true,
	})

	if p.ToolChoice != "required" {
		t.Errorf("tool_choice = %q, want %q", p.ToolChoice, "required")
	}

	content := p.Input[0].Content
	if len(content) != 3 {
		t.Fatalf("content length = %d, want 3 (2 images + 1 text)", len(content))
	}

	// First two should be input_image
	if content[0].Type != "input_image" {
		t.Errorf("content[0].type = %q, want %q", content[0].Type, "input_image")
	}
	if content[0].ImageURL != "data:image/png;base64,abc123" {
		t.Errorf("content[0].image_url wrong")
	}
	if content[1].Type != "input_image" {
		t.Errorf("content[1].type = %q, want %q", content[1].Type, "input_image")
	}

	// Last should be input_text
	if content[2].Type != "input_text" {
		t.Errorf("content[2].type = %q, want %q", content[2].Type, "input_text")
	}

	// Size should be present
	if p.Tools[0].Size != "1024x1024" {
		t.Errorf("tool size = %q, want %q", p.Tools[0].Size, "1024x1024")
	}
}

func TestBuildPayload_SizeOmittedInJSON(t *testing.T) {
	p := BuildPayload(PayloadOptions{
		Model:  "gpt-5.5",
		Prompt: "test",
		Format: "png",
		Size:   "auto",
	})

	data, err := MarshalPayload(p)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Check that the size field is omitted from the JSON
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	tools := raw["tools"].([]interface{})
	tool := tools[0].(map[string]interface{})
	if _, exists := tool["size"]; exists {
		t.Error("size should be omitted from JSON when auto")
	}
}

func TestBuildPayload_Reasoning(t *testing.T) {
	p := BuildPayload(PayloadOptions{
		Model:  "gpt-5.5",
		Prompt: "test",
		Format: "png",
		Size:   "auto",
	})

	if p.Reasoning.Effort != "low" {
		t.Errorf("reasoning.effort = %q, want %q", p.Reasoning.Effort, "low")
	}
	if p.Reasoning.Summary != "auto" {
		t.Errorf("reasoning.summary = %q, want %q", p.Reasoning.Summary, "auto")
	}
	if len(p.Include) != 1 || p.Include[0] != "reasoning.encrypted_content" {
		t.Errorf("include = %v, want [reasoning.encrypted_content]", p.Include)
	}
	if p.Text.Verbosity != "low" {
		t.Errorf("text.verbosity = %q, want %q", p.Text.Verbosity, "low")
	}
}
