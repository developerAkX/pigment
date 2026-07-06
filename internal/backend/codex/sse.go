package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// SSEEvent represents a parsed SSE event.
type SSEEvent struct {
	Type string
	Data json.RawMessage
	Raw  map[string]interface{}
}

// ImageResult holds the final image data from the stream.
type ImageResult struct {
	Base64Data    string
	RevisedPrompt string
	ItemMeta      map[string]interface{}
}

// StreamResult holds the complete result of processing an SSE stream.
type StreamResult struct {
	Image      *ImageResult
	Error      string
	EventTypes []string // sorted, deduplicated
}

// PhaseCallback is called when the generation phase changes.
type PhaseCallback func(phase string, partialCount int)

// ParseSSEStream reads an SSE stream from reader and extracts the image result.
func ParseSSEStream(reader io.Reader, onPhase PhaseCallback) *StreamResult {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line

	var dataLines []string
	eventTypesSeen := make(map[string]bool)
	var result StreamResult
	var errorDetail string
	partialCount := 0

	flushEvent := func() {
		if len(dataLines) == 0 {
			return
		}
		joined := strings.Join(dataLines, "\n")
		dataLines = nil

		if joined == "[DONE]" {
			return
		}

		var evt map[string]interface{}
		if err := json.Unmarshal([]byte(joined), &evt); err != nil {
			return
		}

		evtType, _ := evt["type"].(string)
		if evtType != "" {
			eventTypesSeen[evtType] = true
		}

		switch evtType {
		case "response.image_generation_call.in_progress":
			if onPhase != nil {
				onPhase("queued", 0)
			}

		case "response.image_generation_call.generating":
			if onPhase != nil {
				onPhase("generating", 0)
			}

		case "response.image_generation_call.partial_image":
			partialCount++
			if onPhase != nil {
				onPhase("partial", partialCount)
			}

		case "response.output_item.done":
			item, ok := evt["item"].(map[string]interface{})
			if !ok {
				return
			}
			itemType, _ := item["type"].(string)
			if itemType != "image_generation_call" {
				return
			}
			b64, ok := item["result"].(string)
			if !ok || b64 == "" {
				return
			}

			ir := &ImageResult{
				Base64Data: b64,
				ItemMeta:   make(map[string]interface{}),
			}
			if rp, ok := item["revised_prompt"].(string); ok {
				ir.RevisedPrompt = rp
			}
			// Copy all item fields except result
			for k, v := range item {
				if k != "result" {
					ir.ItemMeta[k] = v
				}
			}
			result.Image = ir

		case "error", "response.failed":
			errorDetail = extractErrorDetail(evt)
		}
	}

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// End of event
			flushEvent()
			continue
		}

		if strings.HasPrefix(line, ":") {
			// Comment/keepalive — ignore
			continue
		}

		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			if len(data) > 0 && data[0] == ' ' {
				data = data[1:]
			}
			dataLines = append(dataLines, data)
			continue
		}

		if strings.HasPrefix(line, "event:") {
			// Tracked but not used for dispatch
			continue
		}

		// id:, retry: — ignored
	}

	// Flush any trailing event
	flushEvent()

	scanErr := scanner.Err()

	// Build sorted event types list
	for t := range eventTypesSeen {
		result.EventTypes = append(result.EventTypes, t)
	}
	sort.Strings(result.EventTypes)

	// Set error message
	if result.Image == nil {
		if scanErr != nil {
			result.Error = fmt.Sprintf(
				"error reading response stream: %v (events seen: %s)",
				scanErr, strings.Join(result.EventTypes, ", "),
			)
		} else if errorDetail != "" {
			result.Error = fmt.Sprintf(
				"backend failed mid-generation: %s (events seen: %s)",
				errorDetail, strings.Join(result.EventTypes, ", "),
			)
		} else {
			result.Error = fmt.Sprintf(
				"no image returned. events seen: %s",
				strings.Join(result.EventTypes, ", "),
			)
		}
	}

	return &result
}

// extractErrorDetail extracts error information from an SSE event per spec precedence.
func extractErrorDetail(evt map[string]interface{}) string {
	// 1. event.response.error.message or event.response.error.code
	if resp, ok := evt["response"].(map[string]interface{}); ok {
		if errObj, ok := resp["error"].(map[string]interface{}); ok {
			if msg, ok := errObj["message"].(string); ok && msg != "" {
				return msg
			}
			if code, ok := errObj["code"].(string); ok && code != "" {
				return code
			}
		}
	}

	// 2. event.message or event.code or event.error (if string)
	if msg, ok := evt["message"].(string); ok && msg != "" {
		return msg
	}
	if code, ok := evt["code"].(string); ok && code != "" {
		return code
	}
	if errStr, ok := evt["error"].(string); ok && errStr != "" {
		return errStr
	}

	// 3. event.error.message (if event.error is an object)
	if errObj, ok := evt["error"].(map[string]interface{}); ok {
		if msg, ok := errObj["message"].(string); ok && msg != "" {
			return msg
		}
	}

	return ""
}
