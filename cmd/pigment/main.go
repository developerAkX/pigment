// pigment is a cross-platform CLI for generating and editing images
// via a ChatGPT subscription OAuth backend.
package main

import (
	"fmt"
	"os"

	"github.com/developerAkX/pigment/internal/cli"
)

func main() {
	rootCmd := cli.NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
