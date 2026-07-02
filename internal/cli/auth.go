package cli

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/developerAkX/pigment/internal/auth"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication (status, login, logout)",
	}

	cmd.AddCommand(newAuthStatusCmd())
	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthLogoutCmd())

	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tokens, err := auth.LoadTokens()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Not authenticated: %v\n", err)
				return err
			}

			fmt.Println("Authentication status:")
			fmt.Printf("  Access token: present (%d chars)\n", len(tokens.AccessToken))

			if tokens.AccountID != "" {
				fmt.Printf("  Account ID:   %s\n", tokens.AccountID)
			} else {
				fmt.Println("  Account ID:   (not set)")
			}

			if tokens.RefreshToken != "" {
				fmt.Println("  Refresh token: present")
			} else {
				fmt.Println("  Refresh token: (not set)")
			}

			if !tokens.LastRefresh.IsZero() {
				age := time.Since(tokens.LastRefresh)
				fmt.Printf("  Last refresh:  %s (%.0f hours ago)\n",
					tokens.LastRefresh.Format("2006-01-02 15:04:05 UTC"),
					age.Hours())
			} else {
				fmt.Println("  Last refresh:  (unknown)")
			}

			return nil
		},
	}
}

func newAuthLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Instructions to authenticate",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			// Check if codex CLI is installed
			_, err := exec.LookPath("codex")
			if err != nil {
				fmt.Println("Codex CLI is not installed.")
				fmt.Println("Install it with: npm i -g @openai/codex")
				fmt.Println("Then run: codex login")
			} else {
				fmt.Println("Codex CLI is installed. Run:")
				fmt.Println("  codex login")
				fmt.Println()
				fmt.Println("This will authenticate with your ChatGPT subscription")
				fmt.Println("and save tokens to ~/.codex/auth.json.")
			}
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Instructions to log out",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("To log out, remove the auth file:")
			fmt.Println("  rm ~/.codex/auth.json")
			fmt.Println()
			fmt.Println("Or use the Codex CLI if it supports logout.")
		},
	}
}
