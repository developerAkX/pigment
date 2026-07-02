package styles

// builtinStyles are the three built-in styles per spec §6.5.
var builtinStyles = map[string]Entry{
	"doodle": {
		Kind: KindStyle,
		Snippet: "drawn as a deliberately crude doodle — thick unsteady outlines, " +
			"wobbly shapes, minimal shading, kindergarten felt-tip-pen energy; " +
			"keep it endearingly rough, not polished",
		Refs: []string{},
	},
	"xiaohei": {
		Kind: KindStyle,
		Snippet: "Ian 'Xiaohei' (\u5c0f\u9ed1) hand-drawn explainer style: " +
			"clean black ink outlines on white background, minimal flat color fills, " +
			"small annotations in a casual handwriting font, whiteboard-diagram feel",
		Refs: []string{},
	},
	"snoopy": {
		Kind: KindStyle,
		Snippet: "Classic mid-20th-century American newspaper comic-strip style " +
			"(think Peanuts / Snoopy): simple clean ink lines, flat limited palette, " +
			"expressive characters with big heads, minimalist backgrounds, " +
			"gentle humor, no halftone dots",
		Refs: []string{},
	},
}

// builtinNames returns the names of all built-in styles.
func builtinNames() []string {
	names := make([]string, 0, len(builtinStyles))
	for k := range builtinStyles {
		names = append(names, k)
	}
	return names
}

// freshDoc creates a brand-new doc with all built-in styles seeded.
func (s *Store) freshDoc() *Doc {
	styles := make(map[string]Entry, len(builtinStyles))
	seeded := make([]string, 0, len(builtinStyles))
	for name, entry := range builtinStyles {
		e := entry
		if e.Refs == nil {
			e.Refs = []string{}
		}
		styles[name] = e
		seeded = append(seeded, name)
	}
	return &Doc{
		Version: 2,
		Default: nil,
		Seeded:  seeded,
		Styles:  styles,
	}
}

// mergeBuiltins merges new built-in styles into an existing doc.
// Returns true if the doc was modified (caller should save).
func (s *Store) mergeBuiltins(doc *Doc) bool {
	changed := false
	bNames := builtinNames()

	// If seeded is nil (legacy), initialize to intersection of builtins and existing styles
	if doc.Seeded == nil {
		doc.Seeded = []string{}
		for _, bn := range bNames {
			if _, exists := doc.Styles[bn]; exists {
				doc.Seeded = append(doc.Seeded, bn)
			}
		}
		changed = true
	}

	seededSet := make(map[string]bool, len(doc.Seeded))
	for _, s := range doc.Seeded {
		seededSet[s] = true
	}

	for _, bn := range bNames {
		if seededSet[bn] {
			continue // already offered
		}
		if _, exists := doc.Styles[bn]; exists {
			continue // user has their own entry with this name
		}
		// Add the built-in
		e := builtinStyles[bn]
		if e.Refs == nil {
			e.Refs = []string{}
		}
		doc.Styles[bn] = e
		changed = true
	}

	// Update seeded to union of old seeded + all current builtin names
	for _, bn := range bNames {
		if !seededSet[bn] {
			doc.Seeded = append(doc.Seeded, bn)
			changed = true
		}
	}
	return changed
}
