//go:build windows

package config

import (
	"os"
)

func processAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Windows, FindProcess always succeeds. Signal(0) doesn't work.
	// Best effort: try to open the process.
	err = p.Signal(os.Interrupt)
	_ = err
	// Assume alive if we can't tell
	return true
}
