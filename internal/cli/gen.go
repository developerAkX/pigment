package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/developerAkX/pigment/internal/auth"
	"github.com/developerAkX/pigment/internal/backend/codex"
	"github.com/developerAkX/pigment/internal/config"
	"github.com/developerAkX/pigment/internal/imagegen"
	"github.com/developerAkX/pigment/internal/output"
	"github.com/developerAkX/pigment/internal/styles"
	"github.com/spf13/cobra"
)

type genFlags struct {
	output       string
	size         string
	format       string
	model        string
	refs         []string
	styleNames   []string
	noStyle      bool
	timeout      int
	stallTimeout int
	noProgress   bool
	jsonOutput   bool
	open         bool
}

func newGenCmd() *cobra.Command {
	f := &genFlags{}

	cmd := &cobra.Command{
		Use:   `gen "<prompt>"`,
		Short: "Generate an image from a text prompt",
		Long: `Generate an image from a text prompt using the ChatGPT codex backend.
Reference images can be provided with -i/--ref for image-to-image generation.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGen(args[0], f)
		},
	}

	cmd.Flags().StringVarP(&f.output, "out", "o", "", "output file path")
	cmd.Flags().StringVar(&f.size, "size", "auto", "image size (auto, 1024x1024, etc.)")
	cmd.Flags().StringVar(&f.format, "format", "png", "output format (png, jpeg, webp)")
	cmd.Flags().StringVar(&f.model, "model", "", "model name (default from PIGMENT_MODEL or gpt-5.5)")
	cmd.Flags().StringArrayVarP(&f.refs, "ref", "i", nil, "reference image path or URL (repeatable)")
	cmd.Flags().StringArrayVar(&f.styleNames, "style", nil, "apply named style(s) (repeatable)")
	cmd.Flags().BoolVar(&f.noStyle, "no-style", false, "suppress all styles for this run")
	cmd.Flags().IntVar(&f.timeout, "timeout", 300, "total timeout in seconds")
	cmd.Flags().IntVar(&f.stallTimeout, "stall-timeout", 120, "stall timeout in seconds")
	cmd.Flags().BoolVar(&f.noProgress, "no-progress", false, "suppress progress output")
	cmd.Flags().BoolVar(&f.jsonOutput, "json", false, "output JSON instead of path")
	cmd.Flags().BoolVar(&f.open, "open", false, "open the result in the default viewer")

	return cmd
}

func runGen(prompt string, f *genFlags) error {
	startTime := time.Now()

	// Validate --style and --no-style mutual exclusivity
	if f.noStyle && len(f.styleNames) > 0 {
		return fmt.Errorf("--style and --no-style are mutually exclusive")
	}

	// Set warn writer
	codex.SetWarnWriter(os.Stderr)

	// Resolve model
	model := f.model
	if model == "" {
		model = config.DefaultModel()
	}

	// Validate format
	switch f.format {
	case "png", "jpeg", "webp":
	default:
		return fmt.Errorf("invalid format %q: must be png, jpeg, or webp", f.format)
	}

	// Progress printer
	progress := output.NewProgress(!f.noProgress)

	// Load auth
	progress.Print("loading credentials")
	tokens, err := auth.LoadTokens()
	if err != nil {
		return err
	}

	// Resolve active styles
	styleStore := styles.NewDefaultStore()
	var activeEntries []styles.ActiveEntry
	if !f.noStyle {
		doc, loadErr := styleStore.Load()
		if loadErr != nil {
			// Non-fatal: proceed without styles
			fmt.Fprintf(os.Stderr, "warning: could not load styles: %v\n", loadErr)
		} else {
			if len(f.styleNames) > 0 {
				// Explicit --style flags
				for _, sn := range f.styleNames {
					entry, err := styleStore.Get(doc, sn)
					if err != nil {
						return err
					}
					activeEntries = append(activeEntries, styles.ActiveEntry{Name: sn, Entry: *entry})
				}
			} else {
				// Use active default set
				activeEntries = styleStore.ActiveStyles(doc)
			}
		}
	}

	// Compose prompt with style snippets
	composedRawPrompt := styles.ComposeSnippets(prompt, activeEntries)

	// Collect refs: character asset refs, ad-hoc --ref (character), style asset refs
	var refs []*imagegen.RefImage

	// 1. Character asset refs (from active styles of kind=character)
	for _, ae := range activeEntries {
		if ae.Entry.Kind != styles.KindCharacter {
			continue
		}
		for _, refFile := range ae.Entry.Refs {
			refPath := filepath.Join(styleStore.AssetDir(ae.Name), refFile)
			ref, err := imagegen.LoadRef(refPath, imagegen.RefKindCharacter)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not load style ref %s/%s: %v\n", ae.Name, refFile, err)
				continue
			}
			ref.Label = ae.Name + "/" + refFile
			refs = append(refs, ref)
		}
	}

	// 2. Ad-hoc --ref flags (treated as character/subject refs)
	if len(f.refs) > 0 {
		progress.Print(fmt.Sprintf("loading %d reference image(s)", len(f.refs)))
		loaded, err := imagegen.LoadRefs(f.refs, imagegen.RefKindCharacter)
		if err != nil {
			return err
		}
		refs = append(refs, loaded...)
	}

	// 3. Style asset refs (from active styles of kind=style)
	for _, ae := range activeEntries {
		if ae.Entry.Kind != styles.KindStyle {
			continue
		}
		for _, refFile := range ae.Entry.Refs {
			refPath := filepath.Join(styleStore.AssetDir(ae.Name), refFile)
			ref, err := imagegen.LoadRef(refPath, imagegen.RefKindStyle)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not load style ref %s/%s: %v\n", ae.Name, refFile, err)
				continue
			}
			ref.Label = ae.Name + "/" + refFile
			refs = append(refs, ref)
		}
	}

	// Enforce ref cap
	if len(refs) > 0 {
		var dropped []string
		refs, dropped = imagegen.EnforceRefCap(refs)
		if len(dropped) > 0 {
			// Always warn, even under --no-progress
			fmt.Fprintf(os.Stderr,
				"warning: more than 4 reference images resolved; attaching the first 4 (character-first), dropped: %s\n",
				joinLabels(dropped))
		}
	}

	// Build data URIs
	var dataURIs []string
	for _, ref := range refs {
		dataURIs = append(dataURIs, ref.DataURI)
	}

	// Count char vs style refs
	charCount := 0
	styleCount := 0
	for _, ref := range refs {
		if ref.Kind == imagegen.RefKindCharacter {
			charCount++
		} else {
			styleCount++
		}
	}

	composedPrompt := imagegen.ComposePrompt(imagegen.PromptOptions{
		RawPrompt:     composedRawPrompt,
		Format:        f.format,
		Size:          f.size,
		CharRefCount:  charCount,
		StyleRefCount: styleCount,
	})

	// Build payload
	payload := codex.BuildPayload(codex.PayloadOptions{
		Model:       model,
		Prompt:      composedPrompt,
		Format:      f.format,
		Size:        f.size,
		RefDataURIs: dataURIs,
		HasRefs:     len(refs) > 0,
	})

	// Acquire concurrency slot
	concurrency := config.CodexConcurrency()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	slot, err := config.AcquireSlot(ctx, "codex", concurrency, func() {
		progress.Print(fmt.Sprintf("waiting for a free codex slot (max %d concurrent codex generation(s))", concurrency))
	})
	if err != nil {
		return fmt.Errorf("interrupted while waiting for slot: %v", err)
	}
	defer slot.Release()

	// Now the timeout budget starts
	totalTimeout := time.Duration(f.timeout) * time.Second
	stallTimeout := time.Duration(f.stallTimeout) * time.Second
	if stallTimeout > totalTimeout {
		stallTimeout = totalTimeout
	}

	genCtx, genCancel := context.WithTimeout(ctx, totalTimeout)
	defer genCancel()

	progress.Print(fmt.Sprintf("generating with %s", model))

	// Generate
	genReq := &codex.GenerateRequest{
		Tokens:       tokens,
		Payload:      payload,
		TotalTimeout: totalTimeout,
		StallTimeout: stallTimeout,
		OnPhase: func(phase string, partialCount int) {
			switch phase {
			case "queued":
				progress.Print("queued")
			case "generating":
				progress.Print("generating")
			case "partial":
				progress.Print(fmt.Sprintf("receiving image (partial %d)", partialCount))
			}
		},
	}

	resp, err := codex.Generate(genCtx, genReq)
	if err != nil {
		return err
	}

	// Resolve output path
	outPath, err := output.ResolveOutputPath(f.output, prompt, f.format)
	if err != nil {
		return err
	}

	// Save image
	if err := output.SaveImage(outPath, resp.ImageBytes); err != nil {
		return fmt.Errorf("failed to save image: %v", err)
	}

	// Record last output (best-effort)
	if absPath, err := filepath.Abs(outPath); err == nil {
		styles.RecordLastOutput(absPath)
	}

	duration := time.Since(startTime)
	progress.Print(fmt.Sprintf("saved to %s (%.1fs)", outPath, duration.Seconds()))

	// Output to stdout
	if f.jsonOutput {
		sizeStr := f.size
		if meta, ok := resp.ItemMeta["size"].(string); ok && meta != "" {
			sizeStr = meta
		}
		output.PrintJSON(&output.JSONOutput{
			Path:       outPath,
			Model:      model,
			Size:       sizeStr,
			Format:     f.format,
			DurationMS: duration.Milliseconds(),
			Prompt:     prompt,
		})
	} else {
		fmt.Println(outPath)
	}

	// Open if requested
	if f.open {
		output.OpenFile(outPath)
	}

	return nil
}

func joinLabels(labels []string) string {
	result := ""
	for i, l := range labels {
		if i > 0 {
			result += ", "
		}
		result += l
	}
	return result
}
