// Package output handles default output paths, file saving, and progress display.
package output

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// DefaultOutputDir is where generated images go by default.
const DefaultOutputDir = "assets/generated"

// Slug generates a filename slug from a prompt.
func Slug(prompt string) string {
	s := strings.ToLower(prompt)
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = s[:60]
	}
	if s == "" {
		s = "image"
	}
	return s
}

// DefaultPath returns the default output path for a generated image.
// It handles auto-numbering for collision avoidance.
func DefaultPath(prompt, format string) string {
	slug := Slug(prompt)
	ext := format // png, jpeg, webp
	dir := DefaultOutputDir

	// First try without number suffix
	path := filepath.Join(dir, fmt.Sprintf("%s.%s", slug, ext))
	if !fileExists(path) {
		return path
	}

	// Auto-number: -2, -3, ...
	for i := 2; ; i++ {
		path = filepath.Join(dir, fmt.Sprintf("%s-%d.%s", slug, i, ext))
		if !fileExists(path) {
			return path
		}
	}
}

// ResolveOutputPath returns the output path to use, creating parent dirs.
func ResolveOutputPath(explicit, prompt, format string) (string, error) {
	var path string
	if explicit != "" {
		path = explicit
	} else {
		path = DefaultPath(prompt, format)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory %s: %v", dir, err)
	}

	// Warn on format/extension mismatch
	if explicit != "" {
		checkFormatMismatch(path, format)
	}

	return path, nil
}

func checkFormatMismatch(path, format string) {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	ext = strings.ToLower(ext)

	// Normalize jpg/jpeg
	if ext == "jpg" {
		ext = "jpeg"
	}
	normFmt := format
	if normFmt == "jpg" {
		normFmt = "jpeg"
	}

	if ext != "" && ext != normFmt {
		fmt.Fprintf(os.Stderr, "warning: --format=%s but %s has .%s extension; writing %s bytes anyway\n",
			format, filepath.Base(path), filepath.Ext(path)[1:], format)
	}
}

// SaveImage writes raw image bytes to the given path.
func SaveImage(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
