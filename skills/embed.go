// Package skills embeds the pigment agent skill files and provides
// installation logic for opencode, claude, and agents skill directories.
//
// The skill directories in this package (pigment-generate, pigment-edit,
// pigment-style) are the single source of truth: they are both embedded
// into the binary via go:embed and discovered by the skills.sh registry
// (`npx skills add developerAkX/pigment`) because they live at
// skills/<name>/SKILL.md in the repository.
package skills

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/developerAkX/pigment/internal/version"
)

//go:embed pigment-generate/SKILL.md pigment-edit/SKILL.md pigment-style/SKILL.md
var skillsFS embed.FS

// Stamp is the comment appended to installed SKILL.md files.
func Stamp() string {
	return fmt.Sprintf("<!-- installed by pigment v%s -->", version.Version)
}

// SkillInfo describes an embedded skill.
type SkillInfo struct {
	Name string
	Dir  string // relative path inside embed (e.g. "pigment-generate")
}

// List returns all embedded skills.
func List() ([]SkillInfo, error) {
	entries, err := fs.ReadDir(skillsFS, ".")
	if err != nil {
		return nil, fmt.Errorf("reading embedded skills: %w", err)
	}
	var out []SkillInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		out = append(out, SkillInfo{
			Name: e.Name(),
			Dir:  e.Name(),
		})
	}
	return out, nil
}

// ReadSkillFile reads a file from the embedded skills FS.
func ReadSkillFile(path string) ([]byte, error) {
	return skillsFS.ReadFile(path)
}

// TargetDir returns the base directory for a given target.
func TargetDir(target string, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	switch target {
	case "opencode":
		return filepath.Join(home, ".config", "opencode", "skills"), nil
	case "claude":
		return filepath.Join(home, ".claude", "skills"), nil
	case "agents":
		return filepath.Join(home, ".agents", "skills"), nil
	default:
		return "", fmt.Errorf("unknown target %q: must be opencode, claude, or agents", target)
	}
}

// Install installs all embedded skills to the target directory.
// Returns a list of installed skill paths. If force is false, it
// refuses to overwrite files that don't contain the pigment stamp.
func Install(target, dirOverride string, force bool) ([]string, error) {
	baseDir, err := TargetDir(target, dirOverride)
	if err != nil {
		return nil, err
	}

	skills, err := List()
	if err != nil {
		return nil, err
	}

	stamp := Stamp()
	var installed []string

	for _, sk := range skills {
		destDir := filepath.Join(baseDir, sk.Name)
		destFile := filepath.Join(destDir, "SKILL.md")

		// Check overwrite protection
		if !force {
			if existing, err := os.ReadFile(destFile); err == nil {
				if !strings.Contains(string(existing), "<!-- installed by pigment v") {
					return nil, fmt.Errorf(
						"%s exists and was not installed by pigment (use --force to overwrite)",
						destFile,
					)
				}
			}
		}

		// Read embedded content
		content, err := ReadSkillFile(sk.Dir + "/SKILL.md")
		if err != nil {
			return nil, fmt.Errorf("reading embedded %s: %w", sk.Name, err)
		}

		// Append stamp
		stamped := string(content) + "\n" + stamp + "\n"

		// Write
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return nil, fmt.Errorf("creating %s: %w", destDir, err)
		}
		if err := os.WriteFile(destFile, []byte(stamped), 0o644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", destFile, err)
		}

		installed = append(installed, destFile)
	}

	return installed, nil
}
