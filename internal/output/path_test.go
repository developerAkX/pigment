package output

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSlug(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple",
			input: "a watercolor cat",
			want:  "a-watercolor-cat",
		},
		{
			name:  "special_chars",
			input: "Hello, World! This is a test.",
			want:  "hello-world-this-is-a-test",
		},
		{
			name:  "leading_trailing_nonalnum",
			input: "---hello---",
			want:  "hello",
		},
		{
			name:  "empty_result",
			input: "!!!",
			want:  "image",
		},
		{
			name:  "empty_string",
			input: "",
			want:  "image",
		},
		{
			name:  "long_prompt",
			input: "this is a very long prompt that should be truncated to sixty characters exactly to fit the slug limit",
			want:  "this-is-a-very-long-prompt-that-should-be-truncated-to-sixty",
		},
		{
			name:  "numbers",
			input: "test 123 image",
			want:  "test-123-image",
		},
		{
			name:  "unicode",
			input: "café résumé",
			want:  "caf-r-sum",
		},
		{
			name:  "consecutive_specials",
			input: "hello---world!!!test",
			want:  "hello-world-test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Slug(tc.input)
			if got != tc.want {
				t.Errorf("Slug(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestDefaultPath_AutoNumbering(t *testing.T) {
	// Create a temp dir and override DefaultOutputDir behavior by working in it
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	// Create the default output directory
	os.MkdirAll("assets/generated", 0755)

	// First call should give no suffix
	p1 := DefaultPath("test image", "png")
	if filepath.Base(p1) != "test-image.png" {
		t.Errorf("first path = %q, want test-image.png", filepath.Base(p1))
	}

	// Create the first file
	os.WriteFile(p1, []byte("fake"), 0644)

	// Second call should give -2 suffix
	p2 := DefaultPath("test image", "png")
	if filepath.Base(p2) != "test-image-2.png" {
		t.Errorf("second path = %q, want test-image-2.png", filepath.Base(p2))
	}

	// Create that too
	os.WriteFile(p2, []byte("fake"), 0644)

	// Third should give -3
	p3 := DefaultPath("test image", "png")
	if filepath.Base(p3) != "test-image-3.png" {
		t.Errorf("third path = %q, want test-image-3.png", filepath.Base(p3))
	}
}

func TestDefaultPath_Formats(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)
	os.MkdirAll("assets/generated", 0755)

	tests := []struct {
		format string
		ext    string
	}{
		{"png", "png"},
		{"jpeg", "jpeg"},
		{"webp", "webp"},
	}

	for _, tc := range tests {
		p := DefaultPath("test", tc.format)
		if filepath.Ext(p) != "."+tc.ext {
			t.Errorf("format=%s: ext = %q, want %q", tc.format, filepath.Ext(p), "."+tc.ext)
		}
	}
}
