package config

import (
	"os"
	"testing"
)

func TestNoColor(t *testing.T) {
	// Save and restore env
	saveAndClear := func(keys ...string) func() {
		saved := make(map[string]string)
		for _, k := range keys {
			saved[k] = os.Getenv(k)
			os.Unsetenv(k)
		}
		return func() {
			for _, k := range keys {
				if v, ok := saved[k]; ok && v != "" {
					os.Setenv(k, v)
				} else {
					os.Unsetenv(k)
				}
			}
		}
	}

	t.Run("no_env", func(t *testing.T) {
		restore := saveAndClear("NO_COLOR", "PIGMENT_NO_COLOR", "TERM")
		defer restore()
		if NoColor() {
			t.Error("expected color when no env vars set")
		}
	})

	t.Run("NO_COLOR", func(t *testing.T) {
		restore := saveAndClear("NO_COLOR", "PIGMENT_NO_COLOR", "TERM")
		defer restore()
		os.Setenv("NO_COLOR", "1")
		if !NoColor() {
			t.Error("expected no color with NO_COLOR=1")
		}
	})

	t.Run("PIGMENT_NO_COLOR", func(t *testing.T) {
		restore := saveAndClear("NO_COLOR", "PIGMENT_NO_COLOR", "TERM")
		defer restore()
		os.Setenv("PIGMENT_NO_COLOR", "true")
		if !NoColor() {
			t.Error("expected no color with PIGMENT_NO_COLOR=true")
		}
	})

	t.Run("TERM_dumb", func(t *testing.T) {
		restore := saveAndClear("NO_COLOR", "PIGMENT_NO_COLOR", "TERM")
		defer restore()
		os.Setenv("TERM", "dumb")
		if !NoColor() {
			t.Error("expected no color with TERM=dumb")
		}
	})

	t.Run("TERM_dumb_uppercase", func(t *testing.T) {
		restore := saveAndClear("NO_COLOR", "PIGMENT_NO_COLOR", "TERM")
		defer restore()
		os.Setenv("TERM", "DUMB")
		if !NoColor() {
			t.Error("expected no color with TERM=DUMB")
		}
	})
}

func TestDefaultModel(t *testing.T) {
	save := os.Getenv("PIGMENT_MODEL")
	defer func() {
		if save != "" {
			os.Setenv("PIGMENT_MODEL", save)
		} else {
			os.Unsetenv("PIGMENT_MODEL")
		}
	}()

	os.Unsetenv("PIGMENT_MODEL")
	if m := DefaultModel(); m != "gpt-5.5" {
		t.Errorf("default model = %q, want gpt-5.5", m)
	}

	os.Setenv("PIGMENT_MODEL", "gpt-4o")
	if m := DefaultModel(); m != "gpt-4o" {
		t.Errorf("model = %q, want gpt-4o", m)
	}
}

func TestCodexConcurrency(t *testing.T) {
	save := os.Getenv("PIGMENT_CODEX_CONCURRENCY")
	defer func() {
		if save != "" {
			os.Setenv("PIGMENT_CODEX_CONCURRENCY", save)
		} else {
			os.Unsetenv("PIGMENT_CODEX_CONCURRENCY")
		}
	}()

	os.Unsetenv("PIGMENT_CODEX_CONCURRENCY")
	if c := CodexConcurrency(); c != 4 {
		t.Errorf("default concurrency = %d, want 4", c)
	}

	os.Setenv("PIGMENT_CODEX_CONCURRENCY", "8")
	if c := CodexConcurrency(); c != 8 {
		t.Errorf("concurrency = %d, want 8", c)
	}

	os.Setenv("PIGMENT_CODEX_CONCURRENCY", "0")
	if c := CodexConcurrency(); c != 0 {
		t.Errorf("concurrency = %d, want 0 (unlimited)", c)
	}

	os.Setenv("PIGMENT_CODEX_CONCURRENCY", "-1")
	if c := CodexConcurrency(); c != 0 {
		t.Errorf("concurrency = %d, want 0 (unlimited) for negative", c)
	}

	os.Setenv("PIGMENT_CODEX_CONCURRENCY", "invalid")
	if c := CodexConcurrency(); c != 4 {
		t.Errorf("concurrency = %d, want 4 (default) for invalid", c)
	}
}

func TestConfigDir(t *testing.T) {
	save := os.Getenv("PIGMENT_CONFIG_DIR")
	defer func() {
		if save != "" {
			os.Setenv("PIGMENT_CONFIG_DIR", save)
		} else {
			os.Unsetenv("PIGMENT_CONFIG_DIR")
		}
	}()

	os.Setenv("PIGMENT_CONFIG_DIR", "/custom/dir")
	if d := ConfigDir(); d != "/custom/dir" {
		t.Errorf("config dir = %q, want /custom/dir", d)
	}

	os.Unsetenv("PIGMENT_CONFIG_DIR")
	d := ConfigDir()
	if d == "" {
		t.Error("config dir should not be empty")
	}
}
