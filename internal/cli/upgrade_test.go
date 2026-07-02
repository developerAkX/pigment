package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/developerAkX/pigment/internal/version"
)

func TestFetchLatestRelease(t *testing.T) {
	expected := ghRelease{
		TagName: "v1.2.3",
		Assets: []ghAsset{
			{Name: "pigment_1.2.3_darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.com/a.tar.gz"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/developerAkX/pigment/releases/latest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	rel, err := fetchLatestRelease(srv.URL, "developerAkX/pigment")
	if err != nil {
		t.Fatalf("fetchLatestRelease error: %v", err)
	}

	if rel.TagName != "v1.2.3" {
		t.Errorf("tag = %q, want v1.2.3", rel.TagName)
	}
	if len(rel.Assets) != 2 {
		t.Errorf("assets = %d, want 2", len(rel.Assets))
	}
}

func TestFetchLatestReleaseAlreadyCurrent(t *testing.T) {
	expected := ghRelease{
		TagName: "v" + version.Version,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	rel, err := fetchLatestRelease(srv.URL, "developerAkX/pigment")
	if err != nil {
		t.Fatalf("fetchLatestRelease error: %v", err)
	}

	latest := rel.TagName[1:] // strip "v"
	if latest != version.Version {
		t.Errorf("expected current version %s, got %s", version.Version, latest)
	}
}

func TestFindChecksum(t *testing.T) {
	checksums := []byte(`abc123  pigment_1.0.0_darwin_arm64.tar.gz
def456  pigment_1.0.0_linux_amd64.tar.gz
`)

	hash, err := findChecksum(checksums, "pigment_1.0.0_darwin_arm64.tar.gz")
	if err != nil {
		t.Fatalf("findChecksum error: %v", err)
	}
	if hash != "abc123" {
		t.Errorf("hash = %q, want abc123", hash)
	}

	_, err = findChecksum(checksums, "nonexistent.tar.gz")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestSha256sum(t *testing.T) {
	data := []byte("hello world")
	got := sha256sum(data)
	want := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if got != want {
		t.Errorf("sha256sum = %q, want %q", got, want)
	}
}
