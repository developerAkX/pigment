package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newEditCmd() *cobra.Command {
	f := &genFlags{}

	cmd := &cobra.Command{
		Use:   `edit "<prompt>"`,
		Short: "Edit an image using reference image(s) and a prompt",
		Long: `Edit an image by providing one or more reference images and a text prompt.
At least one reference image (-i/--ref) is required.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(f.refs) == 0 {
				return fmt.Errorf("edit requires at least one reference image (-i/--ref)")
			}
			return runGen(args[0], f)
		},
	}

	cmd.Flags().StringVarP(&f.output, "out", "o", "", "output file path")
	cmd.Flags().StringVar(&f.size, "size", "auto", "image size (auto, 1024x1024, etc.)")
	cmd.Flags().StringVar(&f.format, "format", "png", "output format (png, jpeg, webp)")
	cmd.Flags().StringVar(&f.model, "model", "", "model name (default from PIGMENT_MODEL or gpt-5.5)")
	cmd.Flags().StringArrayVarP(&f.refs, "ref", "i", nil, "reference image path or URL (repeatable, required)")
	cmd.Flags().StringArrayVar(&f.styleNames, "style", nil, "apply named style(s) (repeatable)")
	cmd.Flags().BoolVar(&f.noStyle, "no-style", false, "suppress all styles for this run")
	cmd.Flags().IntVar(&f.timeout, "timeout", 300, "total timeout in seconds")
	cmd.Flags().IntVar(&f.stallTimeout, "stall-timeout", 120, "stall timeout in seconds")
	cmd.Flags().BoolVar(&f.quiet, "quiet", false, "suppress update notices")
	cmd.Flags().BoolVar(&f.noProgress, "no-progress", false, "suppress progress output")
	cmd.Flags().BoolVar(&f.jsonOutput, "json", false, "output JSON instead of path")
	cmd.Flags().BoolVar(&f.open, "open", false, "open the result in the default viewer")

	_ = cmd.MarkFlagRequired("ref")

	return cmd
}
