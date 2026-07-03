package cli

import (
	"fmt"

	"github.com/developerAkX/pigment/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the pigment version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("pigment %s\n", version.Version)
		},
	}
}
