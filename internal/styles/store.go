// Package styles manages the on-disk style/character library.
// Styles live in styles.json under the pigment config directory,
// with per-style reference images in assets/<name>/.
package styles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/developerAkX/pigment/internal/config"
)

// Kind distinguishes style from character entries.
type Kind string

const (
	KindStyle     Kind = "style"
	KindCharacter Kind = "character"
)

// Entry represents a single style entry in styles.json.
type Entry struct {
	Kind    Kind     `json:"kind"`
	Snippet string   `json:"snippet"`
	Refs    []string `json:"refs"`
}

// Doc is the on-disk styles.json schema (version 2).
type Doc struct {
	Version int              `json:"version"`
	Default []string         `json:"default"`
	Seeded  []string         `json:"seeded"`
	Styles  map[string]Entry `json:"styles"`
}

var validName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// ValidateName checks if a style name is valid.
func ValidateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid style name '%s'; use lowercase letters, digits, '-' or '_', starting with a letter or digit", name)
	}
	return nil
}

// Store provides CRUD operations on the style library.
type Store struct {
	dir string // config directory (contains styles.json and assets/)
}

// NewStore creates a Store using the given config directory.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// NewDefaultStore creates a Store using config.ConfigDir().
func NewDefaultStore() *Store {
	return NewStore(config.ConfigDir())
}

// Dir returns the config directory this store operates on.
func (s *Store) Dir() string {
	return s.dir
}

// StylesPath returns the path to styles.json.
func (s *Store) StylesPath() string {
	return filepath.Join(s.dir, "styles.json")
}

// AssetDir returns the asset directory for a named style.
func (s *Store) AssetDir(name string) string {
	return filepath.Join(s.dir, "assets", name)
}

// Load reads and normalizes styles.json, merging built-in defaults.
// Returns an empty doc if the file does not exist.
func (s *Store) Load() (*Doc, error) {
	data, err := os.ReadFile(s.StylesPath())
	if err != nil {
		if os.IsNotExist(err) {
			doc := s.freshDoc()
			return doc, s.Save(doc)
		}
		return nil, fmt.Errorf("failed to read styles.json: %w", err)
	}

	doc, err := parseDoc(data)
	if err != nil {
		return nil, err
	}

	if s.mergeBuiltins(doc) {
		if saveErr := s.Save(doc); saveErr != nil {
			return nil, saveErr
		}
	}
	return doc, nil
}

// Save persists the doc to styles.json.
func (s *Store) Save(doc *Doc) error {
	doc.Version = 2
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal styles.json: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(s.StylesPath(), data, 0644)
}

// List returns all style names sorted alphabetically.
func (s *Store) List(doc *Doc) []string {
	names := make([]string, 0, len(doc.Styles))
	for k := range doc.Styles {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// Get returns the entry for a named style, or an error if not found.
func (s *Store) Get(doc *Doc, name string) (*Entry, error) {
	e, ok := doc.Styles[name]
	if !ok {
		return nil, fmt.Errorf("unknown style '%s'", name)
	}
	return &e, nil
}

// Add creates or overwrites a style. On overwrite, the asset dir is removed.
func (s *Store) Add(doc *Doc, name string, entry Entry) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if _, exists := doc.Styles[name]; exists {
		// Remove old asset directory on overwrite
		os.RemoveAll(s.AssetDir(name))
	}
	doc.Styles[name] = entry
	return nil
}

// Remove deletes a style and its asset directory.
// Returns true if the style was in the active default set.
func (s *Store) Remove(doc *Doc, name string) (wasDefault bool, err error) {
	if _, ok := doc.Styles[name]; !ok {
		return false, fmt.Errorf("unknown style '%s'", name)
	}
	delete(doc.Styles, name)
	os.RemoveAll(s.AssetDir(name))

	// Remove from default set
	for i, d := range doc.Default {
		if d == name {
			doc.Default = append(doc.Default[:i], doc.Default[i+1:]...)
			wasDefault = true
			break
		}
	}
	// Note: name stays in seeded to prevent re-seeding
	return wasDefault, nil
}

// Use sets the active default set to the given names (in order).
func (s *Store) Use(doc *Doc, names []string) error {
	for _, n := range names {
		if _, ok := doc.Styles[n]; !ok {
			return fmt.Errorf("unknown style '%s'", n)
		}
	}
	doc.Default = names
	return nil
}

// Clear empties the active default set.
func (s *Store) Clear(doc *Doc) {
	doc.Default = nil
}

// Reset removes all asset directories and restores built-in styles.
func (s *Store) Reset(doc *Doc) {
	// Remove entire assets directory
	os.RemoveAll(filepath.Join(s.dir, "assets"))
	// Fresh doc with built-ins
	fresh := s.freshDoc()
	doc.Version = fresh.Version
	doc.Default = fresh.Default
	doc.Seeded = fresh.Seeded
	doc.Styles = fresh.Styles
}

// IsActive returns true if the name is in the active default set.
func (s *Store) IsActive(doc *Doc, name string) bool {
	for _, d := range doc.Default {
		if d == name {
			return true
		}
	}
	return false
}

// ActiveStyles returns the entries in the active default set, in order.
// Unknown names are silently skipped.
func (s *Store) ActiveStyles(doc *Doc) []ActiveEntry {
	var result []ActiveEntry
	for _, name := range doc.Default {
		if e, ok := doc.Styles[name]; ok {
			result = append(result, ActiveEntry{Name: name, Entry: e})
		}
	}
	return result
}

// ActiveEntry pairs a name with its entry for ordered iteration.
type ActiveEntry struct {
	Name  string
	Entry Entry
}

// ComposeSnippets appends style snippets to a raw prompt per spec §6.7.
func ComposeSnippets(rawPrompt string, entries []ActiveEntry) string {
	prompt := strings.TrimRight(rawPrompt, " \t\n\r")
	for _, ae := range entries {
		if ae.Entry.Snippet == "" {
			continue
		}
		// Strip trailing comma or period
		prompt = strings.TrimRight(prompt, " \t\n\r")
		if len(prompt) > 0 {
			last := prompt[len(prompt)-1]
			if last == ',' || last == '.' {
				prompt = prompt[:len(prompt)-1]
			}
		}
		prompt += ", " + ae.Entry.Snippet
	}
	return prompt
}
