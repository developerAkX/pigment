package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestList(t *testing.T) {
	skills, err := List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(skills) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(skills))
	}

	names := make(map[string]bool)
	for _, s := range skills {
		names[s.Name] = true
	}
	for _, want := range []string{"pigment-generate", "pigment-edit", "pigment-style"} {
		if !names[want] {
			t.Errorf("missing skill %q", want)
		}
	}
}

func TestReadSkillFile(t *testing.T) {
	data, err := ReadSkillFile("pigment-generate/SKILL.md")
	if err != nil {
		t.Fatalf("ReadSkillFile error: %v", err)
	}
	if !strings.Contains(string(data), "pigment-generate") {
		t.Error("expected content to contain 'pigment-generate'")
	}
}

func TestInstallToTempDir(t *testing.T) {
	dir := t.TempDir()
	installed, err := Install("opencode", dir, false)
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}
	if len(installed) != 3 {
		t.Fatalf("expected 3 installed, got %d", len(installed))
	}

	// Verify files exist and contain stamp
	for _, path := range installed {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		if !strings.Contains(string(data), "<!-- installed by pigment v") {
			t.Errorf("%s missing stamp", path)
		}
	}
}

func TestInstallOverwriteProtection(t *testing.T) {
	dir := t.TempDir()

	// Create a file that was NOT installed by pigment
	skillDir := filepath.Join(dir, "pigment-generate")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("# custom skill\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Should fail without --force
	_, err := Install("opencode", dir, false)
	if err == nil {
		t.Fatal("expected error for overwrite protection")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should mention --force, got: %v", err)
	}

	// Should succeed with --force
	installed, err := Install("opencode", dir, true)
	if err != nil {
		t.Fatalf("Install with force error: %v", err)
	}
	if len(installed) != 3 {
		t.Fatalf("expected 3 installed with force, got %d", len(installed))
	}
}

func TestInstallOverwriteOwnStamp(t *testing.T) {
	dir := t.TempDir()

	// First install
	_, err := Install("opencode", dir, false)
	if err != nil {
		t.Fatalf("first Install error: %v", err)
	}

	// Second install should succeed (our stamp is present)
	installed, err := Install("opencode", dir, false)
	if err != nil {
		t.Fatalf("second Install error: %v", err)
	}
	if len(installed) != 3 {
		t.Fatalf("expected 3 installed on re-install, got %d", len(installed))
	}
}

func TestStampContainsVersion(t *testing.T) {
	stamp := Stamp()
	if !strings.Contains(stamp, "pigment v") {
		t.Errorf("stamp should contain version: %s", stamp)
	}
}
