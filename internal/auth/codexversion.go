package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const FallbackVersion = "0.130.0"

var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// DetectCodexVersion reads ~/.codex/version.json and returns the effective
// version string using floor logic.
func DetectCodexVersion() string {
	return DetectCodexVersionFrom(VersionFilePath())
}

// DetectCodexVersionFrom reads the given path and applies floor logic.
func DetectCodexVersionFrom(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return FallbackVersion
	}

	var doc struct {
		LatestVersion string `json:"latest_version"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return FallbackVersion
	}

	if !semverRe.MatchString(doc.LatestVersion) {
		return FallbackVersion
	}

	if CompareSemver(doc.LatestVersion, FallbackVersion) > 0 {
		return doc.LatestVersion
	}
	return FallbackVersion
}

// CompareSemver compares two semver strings. Returns >0 if a>b, <0 if a<b, 0 if equal.
func CompareSemver(a, b string) int {
	pa := parseSemverParts(a)
	pb := parseSemverParts(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] - pb[i]
		}
	}
	return 0
}

func parseSemverParts(v string) [3]int {
	parts := strings.SplitN(v, ".", 3)
	var r [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return r
		}
		r[i] = n
	}
	return r
}

// FormatOSInfo returns a platform string like "Mac OS 26.0.1; arm64" or
// "Linux; x86_64" for User-Agent headers.
func FormatOSInfo() string {
	// We'll keep this simple and cross-platform
	return formatOSInfoImpl()
}

// FormatUserAgent builds the User-Agent string.
func FormatUserAgent(codexVersion string) string {
	return fmt.Sprintf("codex_cli_rs/%s (%s) pigment", codexVersion, FormatOSInfo())
}
