//go:build windows

package config

import (
	"errors"
	"syscall"
)

const (
	// PROCESS_QUERY_LIMITED_INFORMATION: query-only access, cannot
	// signal or terminate the target process.
	processQueryLimitedInformation = 0x1000
	// STILL_ACTIVE exit code returned by GetExitCodeProcess for
	// running processes.
	stillActive = 259
)

// processAlive reports whether the process with the given PID is running.
// It opens the process with query-only rights — unlike sending a signal,
// this can never affect the target process.
func processAlive(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		// Access denied means the process exists but we lack rights.
		return errors.Is(err, syscall.ERROR_ACCESS_DENIED)
	}
	defer syscall.CloseHandle(h)

	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		// Handle opened but query failed — assume alive (conservative:
		// a stale lock is better than clobbering a live process's slot).
		return true
	}
	return code == stillActive
}
