package imagegen

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	// For webp decoding
	_ "golang.org/x/image/webp"

	"golang.org/x/image/draw"
)

const (
	RefB64Budget   = 5 * 1024 * 1024  // 5 MB base64 length threshold
	RefMaxEdge     = 2048             // target max dimension for downscale
	RefDownloadMax = 25 * 1024 * 1024 // 25 MB download cap
	RefAttachCap   = 4                // max reference images
)

// RefImage holds a loaded reference image.
type RefImage struct {
	Data    []byte // raw image bytes
	MIME    string // detected MIME type
	Label   string // display label (path or URL)
	DataURI string // data:<mime>;base64,<data>
	Kind    RefKind
}

// LoadRef loads a reference image from a local path or URL.
func LoadRef(source string, kind RefKind) (*RefImage, error) {
	source = expandTilde(source)

	var data []byte
	var label string
	var err error

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		data, err = downloadRef(source)
		if err != nil {
			return nil, err
		}
		label = source
	} else {
		data, err = os.ReadFile(source)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("reference image not found: %s", source)
			}
			return nil, fmt.Errorf("failed to read reference image %s: %v", source, err)
		}
		label = source
	}

	mime := detectMIME(data)
	if mime == "" {
		return nil, fmt.Errorf("unsupported reference image type for '%s' (need PNG, JPEG, or WEBP)", source)
	}

	ref := &RefImage{
		Data:  data,
		MIME:  mime,
		Label: label,
		Kind:  kind,
	}

	// Build data URI, downscale if needed
	b64 := base64.StdEncoding.EncodeToString(data)
	if len(b64) > RefB64Budget {
		downscaled := downscaleImage(data, mime)
		if downscaled != nil {
			ref.Data = downscaled
			ref.MIME = "image/jpeg"
			b64 = base64.StdEncoding.EncodeToString(downscaled)
		}
	}
	ref.DataURI = fmt.Sprintf("data:%s;base64,%s", ref.MIME, b64)

	return ref, nil
}

// LoadRefs loads multiple reference images and enforces the cap.
func LoadRefs(sources []string, kind RefKind) ([]*RefImage, []string, error) {
	var refs []*RefImage
	for _, s := range sources {
		ref, err := LoadRef(s, kind)
		if err != nil {
			return nil, nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil, nil
}

// EnforceRefCap enforces the 4-ref cap, returning kept and dropped labels.
func EnforceRefCap(refs []*RefImage) ([]*RefImage, []string) {
	if len(refs) <= RefAttachCap {
		return refs, nil
	}
	kept := refs[:RefAttachCap]
	var dropped []string
	for _, r := range refs[RefAttachCap:] {
		dropped = append(dropped, r.Label)
	}
	return kept, dropped
}

func downloadRef(url string) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not download reference '%s': %v", url, err)
	}
	req.Header.Set("User-Agent", "codex_cli_rs/0.130.0 pigment")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not download reference '%s': %v", url, err)
	}
	defer resp.Body.Close()

	// Read up to cap + 1 byte
	lr := io.LimitReader(resp.Body, int64(RefDownloadMax+1))
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("could not download reference '%s': %v", url, err)
	}
	if len(data) > RefDownloadMax {
		return nil, fmt.Errorf(
			"reference '%s' is larger than the 25 MB download cap — save it locally and downsize first, then pass the file.",
			url,
		)
	}
	return data, nil
}

// detectMIME detects the MIME type from magic bytes.
func detectMIME(data []byte) string {
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' &&
		data[4] == '\r' && data[5] == '\n' && data[6] == 0x1a && data[7] == '\n' {
		return "image/png"
	}
	if len(data) >= 2 && data[0] == 0xff && data[1] == 0xd8 {
		return "image/jpeg"
	}
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	return ""
}

// downscaleImage downscales an image so its longest edge is RefMaxEdge.
// Returns JPEG bytes or nil if downscaling fails.
func downscaleImage(data []byte, mime string) []byte {
	var src image.Image
	var err error

	switch mime {
	case "image/png":
		src, err = png.Decode(bytes.NewReader(data))
	case "image/jpeg":
		src, err = jpeg.Decode(bytes.NewReader(data))
	default:
		// For webp, use the registered decoder
		src, _, err = image.Decode(bytes.NewReader(data))
	}
	if err != nil {
		return nil
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	maxEdge := w
	if h > maxEdge {
		maxEdge = h
	}
	if maxEdge <= RefMaxEdge {
		return nil // already small enough
	}

	scale := float64(RefMaxEdge) / float64(maxEdge)
	newW := int(float64(w) * scale)
	newH := int(float64(h) * scale)

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil
	}

	return buf.Bytes()
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
