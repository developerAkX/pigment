// Package config provides cross-platform configuration directory resolution
// and concurrency lock management.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ConfigDir returns the pigment configuration directory.
// Priority: $PIGMENT_CONFIG_DIR > XDG_CONFIG_HOME/pigment > platform default.
func ConfigDir() string {
	if d := os.Getenv("PIGMENT_CONFIG_DIR"); d != "" {
		return d
	}
	if runtime.GOOS == "windows" {
		if d := os.Getenv("APPDATA"); d != "" {
			return filepath.Join(d, "pigment")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "AppData", "Roaming", "pigment")
	}
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "pigment")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "pigment")
}

// DefaultModel returns the model from PIGMENT_MODEL or the default.
func DefaultModel() string {
	if m := os.Getenv("PIGMENT_MODEL"); m != "" {
		return m
	}
	return "gpt-5.5"
}

// NoColor returns true if color output should be suppressed.
func NoColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return true
	}
	if os.Getenv("PIGMENT_NO_COLOR") != "" {
		return true
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return true
	}
	return false
}

// CodexConcurrency returns the codex concurrency limit.
func CodexConcurrency() int {
	return parseConcurrencyEnv("PIGMENT_CODEX_CONCURRENCY", 4)
}

func parseConcurrencyEnv(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	if n <= 0 {
		return 0 // unlimited
	}
	return n
}

// DefaultTotalTimeout returns 300s.
func DefaultTotalTimeout() time.Duration { return 300 * time.Second }

// DefaultStallTimeout returns 120s.
func DefaultStallTimeout() time.Duration { return 120 * time.Second }

// LockDir returns the directory for concurrency lock files.
func LockDir() string {
	d := os.TempDir()
	return d
}

// LockSlotPath returns the path for a concurrency slot lock file.
func LockSlotPath(backend string, index int) string {
	return filepath.Join(LockDir(), fmt.Sprintf("pigment-%s-%d.lock", backend, index))
}
