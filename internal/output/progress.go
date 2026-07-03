package output

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/developerAkX/pigment/internal/config"
)

// Progress prints progress messages to stderr with elapsed time prefix.
type Progress struct {
	startTime time.Time
	enabled   bool
	useColor  bool
}

// NewProgress creates a progress printer.
func NewProgress(enabled bool) *Progress {
	useColor := false
	if enabled && !config.NoColor() {
		if fi, err := os.Stderr.Stat(); err == nil {
			useColor = fi.Mode()&os.ModeCharDevice != 0
		}
	}
	return &Progress{
		startTime: time.Now(),
		enabled:   enabled,
		useColor:  useColor,
	}
}

// Print prints a progress line with elapsed time prefix.
func (p *Progress) Print(msg string) {
	if !p.enabled {
		return
	}
	elapsed := time.Since(p.startTime).Seconds()
	prefix := fmt.Sprintf("[%5.1fs]", elapsed)

	if p.useColor {
		isWarning := isWarningMessage(msg)
		dim := "\033[2m"
		yellow := "\033[33m"
		reset := "\033[0m"

		prefix = dim + prefix + reset
		if isWarning {
			msg = yellow + msg + reset
		}
	}

	fmt.Fprintf(os.Stderr, "%s %s\n", prefix, msg)
}

// Warn prints a warning message to stderr regardless of progress enabled state.
func (p *Progress) Warn(msg string) {
	elapsed := time.Since(p.startTime).Seconds()
	prefix := fmt.Sprintf("[%5.1fs]", elapsed)

	if p.useColor {
		dim := "\033[2m"
		yellow := "\033[33m"
		reset := "\033[0m"
		prefix = dim + prefix + reset
		msg = yellow + msg + reset
	}

	fmt.Fprintf(os.Stderr, "%s %s\n", prefix, msg)
}

func isWarningMessage(msg string) bool {
	lower := strings.ToLower(msg)
	warningWords := []string{
		"warning", "unavailable", "could not", "couldn't",
		"failed", "stall", "not connected",
	}
	for _, w := range warningWords {
		if strings.Contains(lower, w) {
			return true
		}
	}
	return false
}
