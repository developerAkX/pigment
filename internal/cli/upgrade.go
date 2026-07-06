package cli

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/developerAkX/pigment/internal/auth"
	"github.com/developerAkX/pigment/internal/version"
	"github.com/spf13/cobra"
)

// upgradeBaseURL is the GitHub API base; override for testing.
var upgradeBaseURL = "https://api.github.com"

const (
	upgradeRepo = "developerAkX/pigment"
	// maxDownloadBytes caps release downloads (binaries are ~10-20MB).
	maxDownloadBytes = 200 << 20
)

// upgradeHTTPClient bounds all upgrade-related HTTP calls so a hung
// server cannot stall the CLI indefinitely.
var upgradeHTTPClient = &http.Client{Timeout: 5 * time.Minute}

func newUpgradeCmd() *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade pigment to the latest version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(checkOnly)
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "only check for updates, don't install")

	return cmd
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func fetchLatestRelease(baseURL, repo string) (*ghRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", baseURL, repo)
	resp, err := upgradeHTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decoding release: %w", err)
	}
	return &rel, nil
}

func runUpgrade(checkOnly bool) error {
	if os.Getenv("PIGMENT_NO_UPDATE_CHECK") != "" {
		fmt.Println("Update checks disabled via PIGMENT_NO_UPDATE_CHECK.")
		return nil
	}

	rel, err := fetchLatestRelease(upgradeBaseURL, upgradeRepo)
	if err != nil {
		return err
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	current := version.Version

	if auth.CompareSemver(latest, current) <= 0 {
		fmt.Printf("pigment %s is already the latest version.\n", current)
		return nil
	}

	fmt.Printf("Current: %s → Latest: %s\n", current, latest)

	if checkOnly {
		return nil
	}

	// Determine asset name
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	assetName := fmt.Sprintf("pigment_%s_%s_%s.%s", latest, goos, goarch, ext)

	// Find asset URL
	var assetURL string
	var checksumsURL string
	for _, a := range rel.Assets {
		if a.Name == assetName {
			assetURL = a.BrowserDownloadURL
		}
		if a.Name == "checksums.txt" {
			checksumsURL = a.BrowserDownloadURL
		}
	}

	if assetURL == "" {
		return fmt.Errorf("no release asset found for %s/%s (%s)", goos, goarch, assetName)
	}

	// Download checksums
	if checksumsURL == "" {
		return fmt.Errorf("checksums.txt not found in release assets")
	}

	fmt.Printf("Downloading %s...\n", assetName)

	checksums, err := downloadBytes(checksumsURL)
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}

	expectedHash, err := findChecksum(checksums, assetName)
	if err != nil {
		return err
	}

	// Download asset
	assetBytes, err := downloadBytes(assetURL)
	if err != nil {
		return fmt.Errorf("downloading asset: %w", err)
	}

	// Verify checksum
	actualHash := sha256sum(assetBytes)
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	fmt.Println("Checksum verified.")

	// Extract binary
	var binaryData []byte
	if ext == "zip" {
		binaryData, err = extractFromZip(assetBytes, "pigment.exe")
	} else {
		binaryData, err = extractFromTarGz(assetBytes, "pigment")
	}
	if err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}

	// Replace current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	if err := atomicReplace(execPath, binaryData); err != nil {
		return fmt.Errorf("replacing executable: %w", err)
	}

	fmt.Printf("Upgraded to pigment %s.\n", latest)
	return nil
}

func downloadBytes(url string) ([]byte, error) {
	resp, err := upgradeHTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDownloadBytes {
		return nil, fmt.Errorf("download from %s exceeds %d bytes", url, maxDownloadBytes)
	}
	return data, nil
}

func sha256sum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func findChecksum(checksums []byte, filename string) (string, error) {
	lines := strings.Split(string(checksums), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == filename {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("checksum for %s not found in checksums.txt", filename)
}

func extractFromTarGz(data []byte, targetName string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		base := filepath.Base(hdr.Name)
		if base == targetName {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("%s not found in archive", targetName)
}

func extractFromZip(data []byte, targetName string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range r.File {
		base := filepath.Base(f.Name)
		if base == targetName {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("%s not found in archive", targetName)
}

func atomicReplace(path string, data []byte) error {
	dir := filepath.Dir(path)

	// Write to temp file
	tmp, err := os.CreateTemp(dir, "pigment-upgrade-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// On Windows, rename the old binary aside first
	if runtime.GOOS == "windows" {
		oldPath := path + ".old"
		os.Remove(oldPath) // best-effort clean old
		if err := os.Rename(path, oldPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("moving old binary aside: %w", err)
		}
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}
