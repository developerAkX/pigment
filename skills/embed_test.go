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
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "pigment" {
		t.Errorf("expected skill %q, got %q", "pigment", skills[0].Name)
	}
}

func TestReadSkillFile(t *testing.T) {
	data, err := ReadSkillFile("pigment/SKILL.md")
	if err != nil {
		t.Fatalf("ReadSkillFile error: %v", err)
	}
	if !strings.Contains(string(data), "name: pigment") {
		t.Error("expected content to contain 'name: pigment'")
	}
}

func TestInstallToTempDir(t *testing.T) {
	dir := t.TempDir()
	installed, err := Install("opencode", dir, false)
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}
	if len(installed) != 1 {
		t.Fatalf("expected 1 installed, got %d", len(installed))
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
	skillDir := filepath.Join(dir, "pigment")
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
	if len(installed) != 1 {
		t.Fatalf("expected 1 installed with force, got %d", len(installed))
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
	if len(installed) != 1 {
		t.Fatalf("expected 1 installed on re-install, got %d", len(installed))
	}
}

func TestInstallRemovesLegacySkills(t *testing.T) {
	dir := t.TempDir()

	stamped := "# old skill\n\n" + Stamp() + "\n"
	for _, name := range legacySkillNames {
		legacyDir := filepath.Join(dir, name)
		if err := os.MkdirAll(legacyDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(legacyDir, "SKILL.md"), []byte(stamped), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A user-authored legacy-named dir without the stamp must survive.
	userDir := filepath.Join(dir, "pigment-style")
	if err := os.WriteFile(filepath.Join(userDir, "SKILL.md"), []byte("# mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Install("opencode", dir, false); err != nil {
		t.Fatalf("Install error: %v", err)
	}

	for _, name := range []string{"pigment-generate", "pigment-edit"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("legacy skill %s should have been removed", name)
		}
	}
	if _, err := os.Stat(filepath.Join(userDir, "SKILL.md")); err != nil {
		t.Errorf("user-authored pigment-style should survive: %v", err)
	}
}

func TestStampContainsVersion(t *testing.T) {
	stamp := Stamp()
	if !strings.Contains(stamp, "pigment v") {
		t.Errorf("stamp should contain version: %s", stamp)
	}
}
