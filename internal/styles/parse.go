package styles

import (
	"encoding/json"
	"fmt"
)

// parseDoc parses and normalizes a styles.json document.
// Handles version 1 (bare strings) and version 2 (entry objects).
func parseDoc(data []byte) (*Doc, error) {
	// First try to parse as a raw JSON object to handle legacy formats.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse styles.json: %w", err)
	}

	doc := &Doc{
		Version: 2,
		Styles:  make(map[string]Entry),
	}

	// Parse version
	if v, ok := raw["version"]; ok {
		var version int
		if err := json.Unmarshal(v, &version); err == nil {
			doc.Version = version
		}
	}

	// Parse default — may be string (v1) or []string (v2)
	if d, ok := raw["default"]; ok {
		var arr []string
		if err := json.Unmarshal(d, &arr); err == nil {
			doc.Default = arr
		} else {
			var single string
			if err := json.Unmarshal(d, &single); err == nil {
				doc.Default = []string{single}
			}
		}
	}

	// Parse seeded
	if s, ok := raw["seeded"]; ok {
		var arr []string
		if err := json.Unmarshal(s, &arr); err == nil {
			doc.Seeded = arr
		}
	}

	// Parse styles — each value may be a string (v1) or Entry object (v2)
	if stylesRaw, ok := raw["styles"]; ok {
		var stylesMap map[string]json.RawMessage
		if err := json.Unmarshal(stylesRaw, &stylesMap); err != nil {
			return nil, fmt.Errorf("failed to parse styles in styles.json: %w", err)
		}

		for name, val := range stylesMap {
			// Try parsing as Entry first (v2)
			var entry Entry
			if err := json.Unmarshal(val, &entry); err == nil && entry.Kind != "" {
				if entry.Refs == nil {
					entry.Refs = []string{}
				}
				doc.Styles[name] = entry
				continue
			}

			// Try as bare string (v1 legacy)
			var snippet string
			if err := json.Unmarshal(val, &snippet); err == nil {
				doc.Styles[name] = Entry{
					Kind:    KindStyle,
					Snippet: snippet,
					Refs:    []string{},
				}
				continue
			}

			return nil, fmt.Errorf("invalid style entry for '%s' in styles.json", name)
		}
	}

	doc.Version = 2
	return doc, nil
}
