package cli

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/developerAkX/pigment/internal/auth"
	"github.com/developerAkX/pigment/internal/backend/codex"
	"github.com/developerAkX/pigment/internal/config"
	"github.com/developerAkX/pigment/internal/version"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check system readiness for image generation",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
}

type checkResult struct {
	ok      bool
	warn    bool
	message string
}

func runDoctor() error {
	useColor := !config.NoColor()
	if useColor {
		if fi, err := os.Stdout.Stat(); err == nil {
			useColor = fi.Mode()&os.ModeCharDevice != 0
		}
	}

	backendReady := true

	// Pigment version
	printCheck(useColor, checkResult{
		ok:      true,
		message: fmt.Sprintf("pigment version %s", version.Version),
	})

	// Go runtime (informational)
	printCheck(useColor, checkResult{
		ok:      true,
		message: "go runtime available",
	})

	// Auth token
	tokens, err := auth.LoadTokens()
	if err != nil {
		printCheck(useColor, checkResult{
			ok:      false,
			message: fmt.Sprintf("codex token: %v", err),
		})
		backendReady = false
	} else {
		msg := "codex token present"
		if tokens.AccountID != "" {
			msg += fmt.Sprintf(" (account: %s)", tokens.AccountID)
		}
		if !tokens.LastRefresh.IsZero() {
			age := time.Since(tokens.LastRefresh)
			msg += fmt.Sprintf(", refreshed %.0fh ago", age.Hours())
		}
		printCheck(useColor, checkResult{ok: true, message: msg})
	}

	// Account ID
	if tokens != nil && tokens.AccountID != "" {
		printCheck(useColor, checkResult{
			ok:      true,
			message: fmt.Sprintf("account id: %s", tokens.AccountID),
		})
	} else if tokens != nil {
		printCheck(useColor, checkResult{
			warn:    true,
			message: "account id not present in auth.json",
		})
	}

	// Version.json
	codexVer := auth.DetectCodexVersion()
	if codexVer == auth.FallbackVersion {
		printCheck(useColor, checkResult{
			warn:    true,
			message: fmt.Sprintf("codex version: using fallback %s (version.json missing or stale)", auth.FallbackVersion),
		})
	} else {
		printCheck(useColor, checkResult{
			ok:      true,
			message: fmt.Sprintf("codex version: %s", codexVer),
		})
	}

	// Network reachability
	u, _ := url.Parse(codex.CodexEndpoint)
	host := u.Hostname()
	port := "443"

	conn, err := net.DialTimeout("tcp", host+":"+port, 5*time.Second)
	if err != nil {
		printCheck(useColor, checkResult{
			ok:      false,
			message: fmt.Sprintf("network: cannot reach %s: %v", host, err),
		})
		backendReady = false
	} else {
		conn.Close()
		printCheck(useColor, checkResult{
			ok:      true,
			message: fmt.Sprintf("network: %s reachable", host),
		})
	}

	// Concurrency setting
	conc := config.CodexConcurrency()
	concMsg := fmt.Sprintf("codex concurrency: %d", conc)
	if conc == 0 {
		concMsg = "codex concurrency: unlimited"
	}
	printCheck(useColor, checkResult{ok: true, message: concMsg})

	// Summary
	fmt.Println()
	if backendReady {
		fmt.Println("codex backend: ready")
		return nil
	}
	return fmt.Errorf("codex backend: not ready")
}

func printCheck(useColor bool, r checkResult) {
	var symbol string
	if useColor {
		green := "\033[32m"
		yellow := "\033[33m"
		red := "\033[31m"
		reset := "\033[0m"

		switch {
		case r.warn:
			symbol = yellow + "⚠" + reset
		case r.ok:
			symbol = green + "✓" + reset
		default:
			symbol = red + "✗" + reset
		}
	} else {
		switch {
		case r.warn:
			symbol = "[warn]"
		case r.ok:
			symbol = "[ok]"
		default:
			symbol = "[fail]"
		}
	}

	fmt.Printf("%s %s\n", symbol, r.message)
}
