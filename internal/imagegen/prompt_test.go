package imagegen

import (
	"strings"
	"testing"
)

func TestComposePrompt_NoRefs(t *testing.T) {
	p := ComposePrompt(PromptOptions{
		RawPrompt:     "a watercolor cat",
		Format:        "png",
		Size:          "auto",
		CharRefCount:  0,
		StyleRefCount: 0,
	})

	if !strings.HasPrefix(p, "Use the image_generation tool to render the following.") {
		t.Errorf("unexpected prefix: %q", p[:60])
	}
	if !strings.Contains(p, "Request: a watercolor cat.") {
		t.Errorf("missing prompt: %q", p)
	}
	if !strings.Contains(p, "Output format: png.") {
		t.Errorf("missing format: %q", p)
	}
	if !strings.HasSuffix(p, "Do not include explanatory text — produce only the image.") {
		t.Errorf("missing suffix: %q", p)
	}
	// Size auto should NOT include size
	if strings.Contains(p, "Size:") {
		t.Errorf("should not contain Size for auto: %q", p)
	}
}

func TestComposePrompt_CharRefsOnly(t *testing.T) {
	p := ComposePrompt(PromptOptions{
		RawPrompt:     "make it blue",
		Format:        "png",
		Size:          "1024x1024",
		CharRefCount:  2,
		StyleRefCount: 0,
	})

	if !strings.Contains(p, "edit the attached reference image(s)") {
		t.Errorf("missing char refs preamble: %q", p)
	}
	if !strings.Contains(p, "Treat the reference as the canonical subject") {
		t.Errorf("missing canonical subject: %q", p)
	}
	if !strings.Contains(p, "Size: 1024x1024.") {
		t.Errorf("missing size: %q", p)
	}
}

func TestComposePrompt_StyleRefsOnly(t *testing.T) {
	p := ComposePrompt(PromptOptions{
		RawPrompt:     "a landscape",
		Format:        "jpeg",
		Size:          "auto",
		CharRefCount:  0,
		StyleRefCount: 1,
	})

	if !strings.Contains(p, "matching the visual style of the attached reference image(s)") {
		t.Errorf("missing style refs preamble: %q", p)
	}
	if !strings.Contains(p, "do not copy their content") {
		t.Errorf("missing do not copy: %q", p)
	}
}

func TestComposePrompt_BothRefs(t *testing.T) {
	p := ComposePrompt(PromptOptions{
		RawPrompt:     "in the forest",
		Format:        "png",
		Size:          "auto",
		CharRefCount:  2,
		StyleRefCount: 1,
	})

	if !strings.Contains(p, "The first 2 attached image(s) show a recurring character") {
		t.Errorf("missing char count: %q", p)
	}
	if !strings.Contains(p, "The remaining 1 attached image(s) are style references") {
		t.Errorf("missing style count: %q", p)
	}
}

func TestComposePrompt_WithSize(t *testing.T) {
	p := ComposePrompt(PromptOptions{
		RawPrompt:     "test",
		Format:        "png",
		Size:          "1536x1024",
		CharRefCount:  0,
		StyleRefCount: 0,
	})

	if !strings.Contains(p, "Size: 1536x1024.") {
		t.Errorf("missing size: %q", p)
	}
}

func TestComposePrompt_SuffixAlways(t *testing.T) {
	tests := []struct {
		name string
		opts PromptOptions
	}{
		{"no_refs", PromptOptions{RawPrompt: "test", Format: "png", Size: "auto"}},
		{"char_refs", PromptOptions{RawPrompt: "test", Format: "png", Size: "auto", CharRefCount: 1}},
		{"style_refs", PromptOptions{RawPrompt: "test", Format: "png", Size: "auto", StyleRefCount: 1}},
		{"both_refs", PromptOptions{RawPrompt: "test", Format: "png", Size: "auto", CharRefCount: 1, StyleRefCount: 1}},
		{"with_size", PromptOptions{RawPrompt: "test", Format: "png", Size: "1024x1024"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := ComposePrompt(tc.opts)
			if !strings.HasSuffix(p, "Do not include explanatory text — produce only the image.") {
				t.Errorf("missing suffix in %q", p)
			}
		})
	}
}
