package codex

import (
	"strings"
	"testing"
)

func TestParseSSEStream_ImageResult(t *testing.T) {
	stream := `event: response.image_generation_call.in_progress
data: {"type": "response.image_generation_call.in_progress"}

event: response.image_generation_call.generating
data: {"type": "response.image_generation_call.generating"}

event: response.image_generation_call.partial_image
data: {"type": "response.image_generation_call.partial_image"}

event: response.output_item.done
data: {"type": "response.output_item.done", "item": {"type": "image_generation_call", "result": "aW1hZ2VieXRlcw==", "revised_prompt": "a cat", "size": "1024x1024"}}

data: [DONE]
`
	var phases []string
	result := ParseSSEStream(strings.NewReader(stream), func(phase string, count int) {
		phases = append(phases, phase)
	})

	if result.Image == nil {
		t.Fatal("expected image result, got nil")
	}
	if result.Image.Base64Data != "aW1hZ2VieXRlcw==" {
		t.Errorf("base64 = %q, want %q", result.Image.Base64Data, "aW1hZ2VieXRlcw==")
	}
	if result.Image.RevisedPrompt != "a cat" {
		t.Errorf("revised_prompt = %q, want %q", result.Image.RevisedPrompt, "a cat")
	}

	// Check that "result" is NOT in ItemMeta
	if _, ok := result.Image.ItemMeta["result"]; ok {
		t.Error("ItemMeta should not contain 'result'")
	}
	// But "size" should be
	if result.Image.ItemMeta["size"] != "1024x1024" {
		t.Errorf("ItemMeta[size] = %v, want %q", result.Image.ItemMeta["size"], "1024x1024")
	}

	if result.Error != "" {
		t.Errorf("unexpected error: %q", result.Error)
	}

	// Check phases
	wantPhases := []string{"queued", "generating", "partial"}
	if len(phases) != len(wantPhases) {
		t.Errorf("phases = %v, want %v", phases, wantPhases)
	}
}

func TestParseSSEStream_Error(t *testing.T) {
	stream := `event: error
data: {"type": "error", "message": "rate limit exceeded", "code": "rate_limit"}

data: [DONE]
`
	result := ParseSSEStream(strings.NewReader(stream), nil)

	if result.Image != nil {
		t.Error("expected no image on error")
	}
	if !strings.Contains(result.Error, "rate limit exceeded") {
		t.Errorf("error = %q, want to contain 'rate limit exceeded'", result.Error)
	}
	if !strings.Contains(result.Error, "events seen:") {
		t.Errorf("error = %q, want to contain 'events seen:'", result.Error)
	}
}

func TestParseSSEStream_ResponseFailed(t *testing.T) {
	stream := `event: response.failed
data: {"type": "response.failed", "response": {"error": {"message": "content policy violation"}}}

data: [DONE]
`
	result := ParseSSEStream(strings.NewReader(stream), nil)

	if result.Image != nil {
		t.Error("expected no image on failure")
	}
	if !strings.Contains(result.Error, "content policy violation") {
		t.Errorf("error = %q, want to contain 'content policy violation'", result.Error)
	}
}

func TestParseSSEStream_NoImage(t *testing.T) {
	stream := `event: response.completed
data: {"type": "response.completed"}

data: [DONE]
`
	result := ParseSSEStream(strings.NewReader(stream), nil)

	if result.Image != nil {
		t.Error("expected no image")
	}
	if !strings.Contains(result.Error, "no image returned") {
		t.Errorf("error = %q, want to contain 'no image returned'", result.Error)
	}
}

func TestParseSSEStream_MultiplePartials(t *testing.T) {
	stream := `data: {"type": "response.image_generation_call.partial_image"}

data: {"type": "response.image_generation_call.partial_image"}

data: {"type": "response.image_generation_call.partial_image"}

data: {"type": "response.output_item.done", "item": {"type": "image_generation_call", "result": "dGVzdA=="}}

data: [DONE]
`
	partialCount := 0
	result := ParseSSEStream(strings.NewReader(stream), func(phase string, count int) {
		if phase == "partial" {
			partialCount = count
		}
	})

	if partialCount != 3 {
		t.Errorf("partial count = %d, want 3", partialCount)
	}
	if result.Image == nil {
		t.Fatal("expected image")
	}
}

func TestParseSSEStream_CommentAndKeepAlive(t *testing.T) {
	stream := `:keepalive
: this is a comment
data: {"type": "response.output_item.done", "item": {"type": "image_generation_call", "result": "dGVzdA=="}}

data: [DONE]
`
	result := ParseSSEStream(strings.NewReader(stream), nil)
	if result.Image == nil {
		t.Fatal("expected image despite comments")
	}
}

func TestParseSSEStream_ErrorPrecedence(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "response_error_message",
			data: `{"type": "response.failed", "response": {"error": {"message": "msg1", "code": "code1"}}, "message": "msg2"}`,
			want: "msg1",
		},
		{
			name: "response_error_code",
			data: `{"type": "response.failed", "response": {"error": {"code": "code1"}}, "message": "msg2"}`,
			want: "code1",
		},
		{
			name: "event_message",
			data: `{"type": "error", "message": "msg2"}`,
			want: "msg2",
		},
		{
			name: "event_code",
			data: `{"type": "error", "code": "code2"}`,
			want: "code2",
		},
		{
			name: "error_string",
			data: `{"type": "error", "error": "err_str"}`,
			want: "err_str",
		},
		{
			name: "error_object_message",
			data: `{"type": "error", "error": {"message": "obj_msg"}}`,
			want: "obj_msg",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stream := "data: " + tc.data + "\n\ndata: [DONE]\n"
			result := ParseSSEStream(strings.NewReader(stream), nil)
			if !strings.Contains(result.Error, tc.want) {
				t.Errorf("error = %q, want to contain %q", result.Error, tc.want)
			}
		})
	}
}

func TestParseSSEStream_EventTypeSorted(t *testing.T) {
	stream := `data: {"type": "z_type"}

data: {"type": "a_type"}

data: {"type": "m_type"}

data: [DONE]
`
	result := ParseSSEStream(strings.NewReader(stream), nil)
	if len(result.EventTypes) != 3 {
		t.Fatalf("event types = %v, want 3 entries", result.EventTypes)
	}
	if result.EventTypes[0] != "a_type" || result.EventTypes[1] != "m_type" || result.EventTypes[2] != "z_type" {
		t.Errorf("event types not sorted: %v", result.EventTypes)
	}
}

func TestParseSSEStream_MultiLineData(t *testing.T) {
	// Test multi-line data accumulation (data split across lines)
	stream := "data: {\"type\": \"response.output_item.done\",\ndata:  \"item\": {\"type\": \"image_generation_call\", \"result\": \"dGVzdA==\"}}\n\ndata: [DONE]\n"
	result := ParseSSEStream(strings.NewReader(stream), nil)
	if result.Image == nil {
		t.Fatal("expected image from multi-line data")
	}
}
