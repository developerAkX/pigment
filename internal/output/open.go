package output

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// OpenFile opens the given file path with the OS default viewer.
// Errors are logged to stderr as warnings, not fatal.
func OpenFile(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	default:
		fmt.Fprintf(os.Stderr, "warning: don't know how to open files on %s\n", runtime.GOOS)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not open %s: %v\n", path, err)
	}
}
