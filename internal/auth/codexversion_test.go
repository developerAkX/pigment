package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectCodexVersionFrom(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "newer_than_floor",
			content: `{"latest_version": "0.135.2"}`,
			want:    "0.135.2",
		},
		{
			name:    "older_than_floor",
			content: `{"latest_version": "0.99.0"}`,
			want:    FallbackVersion,
		},
		{
			name:    "equal_to_floor",
			content: `{"latest_version": "0.130.0"}`,
			want:    FallbackVersion,
		},
		{
			name:    "invalid_semver",
			content: `{"latest_version": "not-a-version"}`,
			want:    FallbackVersion,
		},
		{
			name:    "empty_json",
			content: `{}`,
			want:    FallbackVersion,
		},
		{
			name:    "invalid_json",
			content: `not json`,
			want:    FallbackVersion,
		},
		{
			name:    "much_newer",
			content: `{"latest_version": "1.0.0"}`,
			want:    "1.0.0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "version.json")
			os.WriteFile(p, []byte(tc.content), 0644)

			got := DetectCodexVersionFrom(p)
			if got != tc.want {
				t.Errorf("DetectCodexVersionFrom() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDetectCodexVersionFrom_MissingFile(t *testing.T) {
	got := DetectCodexVersionFrom("/nonexistent/version.json")
	if got != FallbackVersion {
		t.Errorf("expected fallback %q, got %q", FallbackVersion, got)
	}
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		a, b string
		want int // >0, <0, or 0
	}{
		{"0.130.0", "0.130.0", 0},
		{"0.135.2", "0.130.0", 1},
		{"0.99.0", "0.130.0", -1},
		{"1.0.0", "0.130.0", 1},
		{"0.130.1", "0.130.0", 1},
	}

	for _, tc := range tests {
		t.Run(tc.a+"_vs_"+tc.b, func(t *testing.T) {
			got := CompareSemver(tc.a, tc.b)
			switch {
			case tc.want > 0 && got <= 0:
				t.Errorf("CompareSemver(%q, %q) = %d, want >0", tc.a, tc.b, got)
			case tc.want < 0 && got >= 0:
				t.Errorf("CompareSemver(%q, %q) = %d, want <0", tc.a, tc.b, got)
			case tc.want == 0 && got != 0:
				t.Errorf("CompareSemver(%q, %q) = %d, want 0", tc.a, tc.b, got)
			}
		})
	}
}
