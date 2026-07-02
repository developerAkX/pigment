package styles

import (
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestComposeSnippets(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		entries []ActiveEntry
		want    string
	}{
		{
			name:    "no_entries",
			prompt:  "a cat",
			entries: nil,
			want:    "a cat",
		},
		{
			name:   "one_style",
			prompt: "a cat",
			entries: []ActiveEntry{
				{Name: "doodle", Entry: Entry{Snippet: "in doodle style"}},
			},
			want: "a cat, in doodle style",
		},
		{
			name:   "trailing_period",
			prompt: "a cat.",
			entries: []ActiveEntry{
				{Name: "s", Entry: Entry{Snippet: "bold"}},
			},
			want: "a cat, bold",
		},
		{
			name:   "trailing_comma",
			prompt: "a cat,",
			entries: []ActiveEntry{
				{Name: "s", Entry: Entry{Snippet: "bold"}},
			},
			want: "a cat, bold",
		},
		{
			name:   "multiple_styles",
			prompt: "a cat",
			entries: []ActiveEntry{
				{Name: "a", Entry: Entry{Snippet: "bold"}},
				{Name: "b", Entry: Entry{Snippet: "blue"}},
			},
			want: "a cat, bold, blue",
		},
		{
			name:   "empty_snippet_skipped",
			prompt: "a cat",
			entries: []ActiveEntry{
				{Name: "a", Entry: Entry{Snippet: ""}},
				{Name: "b", Entry: Entry{Snippet: "bold"}},
			},
			want: "a cat, bold",
		},
		{
			name:   "trailing_whitespace",
			prompt: "a cat   ",
			entries: []ActiveEntry{
				{Name: "s", Entry: Entry{Snippet: "bold"}},
			},
			want: "a cat, bold",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComposeSnippets(tc.prompt, tc.entries)
			if got != tc.want {
				t.Errorf("ComposeSnippets(%q, ...) = %q, want %q", tc.prompt, got, tc.want)
			}
		})
	}
}

// createTestPNG creates a small valid PNG file at the given path.
func createTestPNG(t *testing.T, path string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestRefCopyAndAddRef(t *testing.T) {
	store := newTestStore(t)
	doc, _ := store.Load()

	// Add a style
	store.Add(doc, "teststyle", Entry{Kind: KindStyle, Snippet: "test", Refs: []string{}})

	// Create a test PNG
	srcPNG := filepath.Join(t.TempDir(), "test.png")
	createTestPNG(t, srcPNG)

	// AddRef
	filename, err := store.AddRef(doc, "teststyle", srcPNG)
	if err != nil {
		t.Fatalf("AddRef() = %v", err)
	}
	if filename != "ref-1.png" {
		t.Errorf("filename = %q, want ref-1.png", filename)
	}

	e := doc.Styles["teststyle"]
	if len(e.Refs) != 1 || e.Refs[0] != "ref-1.png" {
		t.Errorf("refs = %v, want [ref-1.png]", e.Refs)
	}

	// Verify file exists in asset dir
	refPath := filepath.Join(store.AssetDir("teststyle"), "ref-1.png")
	if _, err := os.Stat(refPath); err != nil {
		t.Errorf("ref file not found: %v", err)
	}

	// AddRef again
	srcPNG2 := filepath.Join(t.TempDir(), "test2.png")
	createTestPNG(t, srcPNG2)
	filename2, err := store.AddRef(doc, "teststyle", srcPNG2)
	if err != nil {
		t.Fatalf("AddRef() = %v", err)
	}
	if filename2 != "ref-2.png" {
		t.Errorf("filename2 = %q, want ref-2.png", filename2)
	}
	e2 := doc.Styles["teststyle"]
	if len(e2.Refs) != 2 {
		t.Errorf("refs len = %d, want 2", len(e2.Refs))
	}
}

func TestRemoveRef(t *testing.T) {
	store := newTestStore(t)
	doc, _ := store.Load()

	store.Add(doc, "reftest", Entry{Kind: KindStyle, Snippet: "s", Refs: []string{}})

	srcPNG := filepath.Join(t.TempDir(), "a.png")
	createTestPNG(t, srcPNG)
	store.AddRef(doc, "reftest", srcPNG)

	// Remove the ref
	err := store.RemoveRef(doc, "reftest", "ref-1.png")
	if err != nil {
		t.Fatalf("RemoveRef() = %v", err)
	}
	e := doc.Styles["reftest"]
	if len(e.Refs) != 0 {
		t.Errorf("refs after remove = %v, want empty", e.Refs)
	}

	// Remove nonexistent ref
	err = store.RemoveRef(doc, "reftest", "nofile.png")
	if err == nil {
		t.Error("RemoveRef(nofile) = nil, want error")
	}

	// Remove ref from unknown style
	err = store.RemoveRef(doc, "unknown", "ref-1.png")
	if err == nil {
		t.Error("RemoveRef(unknown) = nil, want error")
	}
}

func TestAddRefToUnknownStyle(t *testing.T) {
	store := newTestStore(t)
	doc, _ := store.Load()

	_, err := store.AddRef(doc, "nonexistent", "/dev/null")
	if err == nil {
		t.Error("AddRef(nonexistent) = nil, want error")
	}
}

func TestFromLast(t *testing.T) {
	dir := t.TempDir()

	// No last-output yet
	_, err := ResolveFromLast(dir)
	if err == nil {
		t.Error("ResolveFromLast() = nil, want error when no file")
	}

	// Create a test image and record it
	imgPath := filepath.Join(dir, "gen.png")
	createTestPNG(t, imgPath)
	RecordLastOutputTo(dir, imgPath)

	// Should resolve
	got, err := ResolveFromLast(dir)
	if err != nil {
		t.Fatalf("ResolveFromLast() = %v", err)
	}
	if got != imgPath {
		t.Errorf("got = %q, want %q", got, imgPath)
	}

	// Delete the image, should report gone
	os.Remove(imgPath)
	_, err = ResolveFromLast(dir)
	if err == nil {
		t.Error("ResolveFromLast() = nil, want error when file gone")
	}
}

func TestRefOrderingCombined(t *testing.T) {
	// This tests the ordering logic: char asset refs first, then ad-hoc, then style asset refs.
	// We simulate the ordering by building refs and checking the Kind assignments.
	store := newTestStore(t)
	doc, _ := store.Load()

	// Add a character style with refs
	store.Add(doc, "mychar", Entry{Kind: KindCharacter, Snippet: "a robot", Refs: []string{}})
	charPNG := filepath.Join(t.TempDir(), "char.png")
	createTestPNG(t, charPNG)
	store.AddRef(doc, "mychar", charPNG)

	// Add a style style with refs
	store.Add(doc, "mystyle", Entry{Kind: KindStyle, Snippet: "ink", Refs: []string{}})
	stylePNG := filepath.Join(t.TempDir(), "style.png")
	createTestPNG(t, stylePNG)
	store.AddRef(doc, "mystyle", stylePNG)

	// Simulate active set ordering: mychar, mystyle
	store.Use(doc, []string{"mychar", "mystyle"})

	active := store.ActiveStyles(doc)
	if len(active) != 2 {
		t.Fatalf("active = %d, want 2", len(active))
	}
	if active[0].Name != "mychar" || active[0].Entry.Kind != KindCharacter {
		t.Errorf("first active = %v, want mychar/character", active[0])
	}
	if active[1].Name != "mystyle" || active[1].Entry.Kind != KindStyle {
		t.Errorf("second active = %v, want mystyle/style", active[1])
	}
}

func TestLastOutputJSON(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "img.png")
	createTestPNG(t, imgPath)
	RecordLastOutputTo(dir, imgPath)

	data, err := os.ReadFile(filepath.Join(dir, "last-output.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lo LastOutput
	if err := json.Unmarshal(data, &lo); err != nil {
		t.Fatal(err)
	}
	if lo.Path != imgPath {
		t.Errorf("path = %q, want %q", lo.Path, imgPath)
	}
	if lo.Ts <= 0 {
		t.Errorf("ts = %f, want > 0", lo.Ts)
	}
}
