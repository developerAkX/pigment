package styles

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("PIGMENT_CONFIG_DIR", dir)
	return NewStore(dir)
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"doodle", true},
		{"my-style", true},
		{"my_style", true},
		{"x123", true},
		{"1start", true},
		{"", false},
		{"-start", false},
		{"_start", false},
		{"UPPER", false},
		{"has space", false},
		{"special!", false},
		{"a.b", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateName(tc.name)
			if tc.valid && err != nil {
				t.Errorf("ValidateName(%q) = %v, want nil", tc.name, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("ValidateName(%q) = nil, want error", tc.name)
			}
		})
	}
}

func TestCRUDRoundTrip(t *testing.T) {
	store := newTestStore(t)

	// Load creates fresh doc with built-ins
	doc, err := store.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if doc.Version != 2 {
		t.Errorf("version = %d, want 2", doc.Version)
	}
	if len(doc.Styles) != 3 {
		t.Errorf("initial styles = %d, want 3 builtins", len(doc.Styles))
	}
	for _, bn := range []string{"doodle", "xiaohei", "snoopy"} {
		if _, ok := doc.Styles[bn]; !ok {
			t.Errorf("missing builtin %q", bn)
		}
	}

	// Add a custom style
	err = store.Add(doc, "custom", Entry{Kind: KindCharacter, Snippet: "a blue robot", Refs: []string{}})
	if err != nil {
		t.Fatalf("Add() = %v", err)
	}
	if err := store.Save(doc); err != nil {
		t.Fatalf("Save() = %v", err)
	}

	// Reload and verify
	doc2, err := store.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if len(doc2.Styles) != 4 {
		t.Errorf("styles after add = %d, want 4", len(doc2.Styles))
	}
	e, err := store.Get(doc2, "custom")
	if err != nil {
		t.Fatalf("Get(custom) = %v", err)
	}
	if e.Kind != KindCharacter {
		t.Errorf("kind = %q, want character", e.Kind)
	}
	if e.Snippet != "a blue robot" {
		t.Errorf("snippet = %q, want 'a blue robot'", e.Snippet)
	}

	// Remove
	wasDefault, err := store.Remove(doc2, "custom")
	if err != nil {
		t.Fatalf("Remove() = %v", err)
	}
	if wasDefault {
		t.Error("wasDefault = true, want false")
	}
	if err := store.Save(doc2); err != nil {
		t.Fatalf("Save() = %v", err)
	}

	// Reload and verify removal
	doc3, err := store.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if _, ok := doc3.Styles["custom"]; ok {
		t.Error("custom should be removed")
	}
}

func TestUnknownStyleError(t *testing.T) {
	store := newTestStore(t)
	doc, err := store.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	_, err = store.Get(doc, "nonexistent")
	if err == nil {
		t.Error("Get(nonexistent) = nil, want error")
	}
}

func TestUseAndClear(t *testing.T) {
	store := newTestStore(t)
	doc, err := store.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	// Use
	if err := store.Use(doc, []string{"doodle", "snoopy"}); err != nil {
		t.Fatalf("Use() = %v", err)
	}
	if len(doc.Default) != 2 {
		t.Errorf("default len = %d, want 2", len(doc.Default))
	}
	if doc.Default[0] != "doodle" || doc.Default[1] != "snoopy" {
		t.Errorf("default = %v, want [doodle snoopy]", doc.Default)
	}
	if !store.IsActive(doc, "doodle") {
		t.Error("doodle should be active")
	}
	if store.IsActive(doc, "xiaohei") {
		t.Error("xiaohei should not be active")
	}

	// Clear
	store.Clear(doc)
	if len(doc.Default) != 0 {
		t.Errorf("default after clear = %v, want empty", doc.Default)
	}

	// Use unknown should fail
	err = store.Use(doc, []string{"nonexistent"})
	if err == nil {
		t.Error("Use(nonexistent) = nil, want error")
	}
}

func TestRemoveFromDefault(t *testing.T) {
	store := newTestStore(t)
	doc, err := store.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	// Set default to doodle
	if err := store.Use(doc, []string{"doodle"}); err != nil {
		t.Fatalf("Use() = %v", err)
	}

	// Remove doodle
	wasDefault, err := store.Remove(doc, "doodle")
	if err != nil {
		t.Fatalf("Remove() = %v", err)
	}
	if !wasDefault {
		t.Error("wasDefault = false, want true")
	}
	if len(doc.Default) != 0 {
		t.Errorf("default after remove = %v, want empty", doc.Default)
	}
}

func TestResetSemantics(t *testing.T) {
	store := newTestStore(t)
	doc, err := store.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	// Add custom style and set default
	store.Add(doc, "custom", Entry{Kind: KindStyle, Snippet: "test", Refs: []string{}})
	store.Use(doc, []string{"custom"})
	store.Save(doc)

	// Reset
	store.Reset(doc)
	store.Save(doc)

	// Verify only builtins remain
	if len(doc.Styles) != 3 {
		t.Errorf("styles after reset = %d, want 3", len(doc.Styles))
	}
	if _, ok := doc.Styles["custom"]; ok {
		t.Error("custom should be gone after reset")
	}
	if len(doc.Default) != 0 {
		t.Errorf("default after reset = %v, want empty", doc.Default)
	}
}

func TestBuiltinMerge_NewBuiltinAdded(t *testing.T) {
	store := newTestStore(t)

	// Write a minimal v2 doc that only has "doodle", seeded only "doodle"
	doc := &Doc{
		Version: 2,
		Seeded:  []string{"doodle"},
		Styles: map[string]Entry{
			"doodle": {Kind: KindStyle, Snippet: "test doodle", Refs: []string{}},
		},
	}
	if err := store.Save(doc); err != nil {
		t.Fatalf("Save() = %v", err)
	}

	// Load should merge xiaohei and snoopy as new builtins
	doc2, err := store.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if _, ok := doc2.Styles["xiaohei"]; !ok {
		t.Error("xiaohei should be merged")
	}
	if _, ok := doc2.Styles["snoopy"]; !ok {
		t.Error("snoopy should be merged")
	}
	// doodle should keep user's custom snippet
	if doc2.Styles["doodle"].Snippet != "test doodle" {
		t.Errorf("doodle snippet = %q, want 'test doodle'", doc2.Styles["doodle"].Snippet)
	}
}

func TestBuiltinMerge_DeletedNotReseeded(t *testing.T) {
	store := newTestStore(t)

	// Load default (seeds all three builtins)
	doc, _ := store.Load()

	// Remove doodle (stays in seeded)
	store.Remove(doc, "doodle")
	store.Save(doc)

	// Reload: doodle should NOT reappear
	doc2, _ := store.Load()
	if _, ok := doc2.Styles["doodle"]; ok {
		t.Error("deleted builtin should not reappear")
	}
}

func TestLegacyV1Parse(t *testing.T) {
	store := newTestStore(t)

	// Write a v1-style file
	v1 := map[string]interface{}{
		"default": "mystyle",
		"styles": map[string]interface{}{
			"mystyle": "bold ink drawing",
		},
	}
	data, _ := json.Marshal(v1)
	os.MkdirAll(store.dir, 0755)
	os.WriteFile(store.StylesPath(), data, 0644)

	doc, err := store.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	// Check normalized to v2
	e, ok := doc.Styles["mystyle"]
	if !ok {
		t.Fatal("mystyle missing")
	}
	if e.Kind != KindStyle {
		t.Errorf("kind = %q, want style", e.Kind)
	}
	if e.Snippet != "bold ink drawing" {
		t.Errorf("snippet = %q", e.Snippet)
	}
	if len(doc.Default) != 1 || doc.Default[0] != "mystyle" {
		t.Errorf("default = %v, want [mystyle]", doc.Default)
	}
}

func TestList(t *testing.T) {
	store := newTestStore(t)
	doc, _ := store.Load()
	names := store.List(doc)
	if len(names) != 3 {
		t.Errorf("list len = %d, want 3", len(names))
	}
	// Should be sorted
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("not sorted: %v", names)
		}
	}
}

func TestActiveStyles(t *testing.T) {
	store := newTestStore(t)
	doc, _ := store.Load()

	store.Use(doc, []string{"snoopy", "doodle"})
	active := store.ActiveStyles(doc)
	if len(active) != 2 {
		t.Fatalf("active len = %d, want 2", len(active))
	}
	if active[0].Name != "snoopy" {
		t.Errorf("first active = %q, want snoopy", active[0].Name)
	}
	if active[1].Name != "doodle" {
		t.Errorf("second active = %q, want doodle", active[1].Name)
	}
}

func TestOverwriteRemovesAssetDir(t *testing.T) {
	store := newTestStore(t)
	doc, _ := store.Load()

	// Create an asset dir with a file
	assetDir := store.AssetDir("test-style")
	os.MkdirAll(assetDir, 0755)
	os.WriteFile(filepath.Join(assetDir, "old-ref.png"), []byte("old"), 0644)

	store.Add(doc, "test-style", Entry{Kind: KindStyle, Snippet: "first", Refs: []string{"old-ref.png"}})
	store.Save(doc)

	// Overwrite
	store.Add(doc, "test-style", Entry{Kind: KindStyle, Snippet: "second", Refs: []string{}})
	store.Save(doc)

	// Old asset dir should be gone
	if _, err := os.Stat(filepath.Join(assetDir, "old-ref.png")); !os.IsNotExist(err) {
		t.Error("old ref should be deleted on overwrite")
	}
}
