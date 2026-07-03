package styles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/developerAkX/pigment/internal/config"
)

// LastOutput is the recorded last generation output (last-output.json).
type LastOutput struct {
	Path string  `json:"path"`
	Ts   float64 `json:"ts"`
}

// lastOutputPath returns the path to last-output.json.
func lastOutputPath(dir string) string {
	return filepath.Join(dir, "last-output.json")
}

// RecordLastOutput writes the last generation output path.
// Best-effort: errors are silently ignored.
func RecordLastOutput(absPath string) {
	dir := config.ConfigDir()
	_ = os.MkdirAll(dir, 0755)
	lo := LastOutput{
		Path: absPath,
		Ts:   float64(time.Now().UnixMilli()) / 1000.0,
	}
	data, err := json.Marshal(lo)
	if err != nil {
		return
	}
	_ = os.WriteFile(lastOutputPath(dir), data, 0644)
}

// RecordLastOutputTo writes to a specific config dir (for testing).
func RecordLastOutputTo(dir string, absPath string) {
	_ = os.MkdirAll(dir, 0755)
	lo := LastOutput{
		Path: absPath,
		Ts:   float64(time.Now().UnixMilli()) / 1000.0,
	}
	data, err := json.Marshal(lo)
	if err != nil {
		return
	}
	_ = os.WriteFile(lastOutputPath(dir), data, 0644)
}

// ResolveFromLast reads the last-output.json and returns the path.
func ResolveFromLast(dir string) (string, error) {
	data, err := os.ReadFile(lastOutputPath(dir))
	if err != nil {
		return "", fmt.Errorf("--from-last: no recently generated image was recorded yet. Generate one first, or pass an explicit --ref <path>")
	}
	var lo LastOutput
	if err := json.Unmarshal(data, &lo); err != nil || lo.Path == "" {
		return "", fmt.Errorf("--from-last: no recently generated image was recorded yet. Generate one first, or pass an explicit --ref <path>")
	}
	if _, err := os.Stat(lo.Path); err != nil {
		return "", fmt.Errorf("--from-last: the last generated image is gone (%s). Pass an explicit --ref <path> instead", lo.Path)
	}
	return lo.Path, nil
}
