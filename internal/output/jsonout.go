package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSONOutput is the --json mode output.
type JSONOutput struct {
	Path       string `json:"path"`
	Model      string `json:"model"`
	Size       string `json:"size"`
	Format     string `json:"format"`
	DurationMS int64  `json:"duration_ms"`
	Prompt     string `json:"prompt"`
}

// PrintJSON writes the JSON output to stdout.
func PrintJSON(out *JSONOutput) {
	data, err := json.Marshal(out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to marshal JSON output: %v\n", err)
		return
	}
	fmt.Println(string(data))
}
