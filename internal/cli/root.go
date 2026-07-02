// Package cli defines the cobra command tree for pigment.
package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root pigment command.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "pigment",
		Short: "Generate and edit images via ChatGPT subscription",
		Long: `pigment generates and edits images using the ChatGPT codex backend.
Requires a ChatGPT subscription. Authenticate via codex login.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(newGenCmd())
	rootCmd.AddCommand(newEditCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newStyleCmd())
	rootCmd.AddCommand(newSkillCmd())
	rootCmd.AddCommand(newUpgradeCmd())

	return rootCmd
}
