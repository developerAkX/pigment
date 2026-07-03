// Package imagegen orchestrates image generation: prompt building,
// reference image loading, and calling the backend.
package imagegen

import "fmt"

// RefKind distinguishes character/subject refs from style refs.
type RefKind int

const (
	RefKindCharacter RefKind = iota
	RefKindStyle
)

// PromptOptions holds parameters for composing the backend prompt text.
type PromptOptions struct {
	RawPrompt     string
	Format        string // png, jpeg, webp
	Size          string // "auto" or "WxH"
	CharRefCount  int    // number of character/subject reference images
	StyleRefCount int    // number of style reference images
}

// ComposePrompt builds the full prompt text sent to the backend.
func ComposePrompt(opts PromptOptions) string {
	totalRefs := opts.CharRefCount + opts.StyleRefCount
	format := opts.Format

	var preamble string
	switch {
	case totalRefs == 0:
		// No refs
		preamble = fmt.Sprintf(
			"Use the image_generation tool to render the following. Request: %s. Output format: %s.",
			opts.RawPrompt, format,
		)
	case opts.CharRefCount > 0 && opts.StyleRefCount == 0:
		// Character refs only
		preamble = fmt.Sprintf(
			"Use the image_generation tool to edit the attached reference image(s). "+
				"Treat the reference as the canonical subject — reproduce its exact pattern, colours, and texture faithfully; "+
				"do not redesign or stylise the subject itself. Request: %s. Output format: %s.",
			opts.RawPrompt, format,
		)
	case opts.CharRefCount == 0 && opts.StyleRefCount > 0:
		// Style refs only
		preamble = fmt.Sprintf(
			"Use the image_generation tool to render the following, matching the visual style of the attached reference image(s); "+
				"do not copy their content. Request: %s. Output format: %s.",
			opts.RawPrompt, format,
		)
	default:
		// Both char and style refs
		preamble = fmt.Sprintf(
			"Use the image_generation tool. The first %d attached image(s) show a recurring character — "+
				"reproduce them faithfully as the subject. The remaining %d attached image(s) are style references — "+
				"match their aesthetic, not their content. Request: %s. Output format: %s.",
			opts.CharRefCount, opts.StyleRefCount, opts.RawPrompt, format,
		)
	}

	// Size hint (if not auto)
	if opts.Size != "auto" && opts.Size != "" {
		preamble += fmt.Sprintf(" Size: %s.", opts.Size)
	}

	// Always append the suffix
	preamble += " Do not include explanatory text — produce only the image."

	return preamble
}
