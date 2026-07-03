package styles

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/developerAkX/pigment/internal/imagegen"
)

// CopyRefImage validates, optionally downsizes, and copies an image
// into the style's asset directory. Returns the filename (e.g. "ref-1.png").
func (s *Store) CopyRefImage(name string, source string, nextIndex int) (string, error) {
	ref, err := imagegen.LoadRef(source, imagegen.RefKindCharacter)
	if err != nil {
		return "", err
	}

	// Determine extension from MIME
	ext := mimeToExt(ref.MIME)

	filename := fmt.Sprintf("ref-%d.%s", nextIndex, ext)
	assetDir := s.AssetDir(name)
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create asset dir: %w", err)
	}

	dest := filepath.Join(assetDir, filename)
	if err := os.WriteFile(dest, ref.Data, 0644); err != nil {
		return "", fmt.Errorf("failed to write ref image: %w", err)
	}

	return filename, nil
}

// AddRef appends a reference image to an existing style.
func (s *Store) AddRef(doc *Doc, name string, source string) (string, error) {
	entry, ok := doc.Styles[name]
	if !ok {
		return "", fmt.Errorf("unknown style '%s'", name)
	}

	nextIndex := len(entry.Refs) + 1
	filename, err := s.CopyRefImage(name, source, nextIndex)
	if err != nil {
		return "", err
	}

	entry.Refs = append(entry.Refs, filename)
	doc.Styles[name] = entry
	return filename, nil
}

// RemoveRef removes a reference by filename from a style.
func (s *Store) RemoveRef(doc *Doc, name string, filename string) error {
	entry, ok := doc.Styles[name]
	if !ok {
		return fmt.Errorf("unknown style '%s'", name)
	}

	found := false
	newRefs := make([]string, 0, len(entry.Refs))
	for _, r := range entry.Refs {
		if r == filename {
			found = true
			continue
		}
		newRefs = append(newRefs, r)
	}
	if !found {
		return fmt.Errorf("ref '%s' not found in style '%s'", filename, name)
	}

	entry.Refs = newRefs
	doc.Styles[name] = entry

	// Best-effort delete the file
	os.Remove(filepath.Join(s.AssetDir(name), filename))
	return nil
}

// ResolveRefPaths returns absolute paths for a style's ref images.
func (s *Store) ResolveRefPaths(name string, refs []string) []string {
	assetDir := s.AssetDir(name)
	paths := make([]string, len(refs))
	for i, r := range refs {
		paths[i] = filepath.Join(assetDir, r)
	}
	return paths
}

func mimeToExt(mime string) string {
	switch mime {
	case "image/png":
		return "png"
	case "image/jpeg":
		return "jpeg"
	case "image/webp":
		return "webp"
	default:
		return "png"
	}
}
