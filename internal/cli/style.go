package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/developerAkX/pigment/internal/styles"
	"github.com/spf13/cobra"
)

func newStyleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "style",
		Aliases: []string{"styles"},
		Short:   "Manage the style/character library",
	}

	cmd.AddCommand(newStyleListCmd())
	cmd.AddCommand(newStyleShowCmd())
	cmd.AddCommand(newStyleAddCmd())
	cmd.AddCommand(newStyleAddRefCmd())
	cmd.AddCommand(newStyleRmRefCmd())
	cmd.AddCommand(newStyleRmCmd())
	cmd.AddCommand(newStyleUseCmd())
	cmd.AddCommand(newStyleClearCmd())
	cmd.AddCommand(newStyleResetCmd())

	return cmd
}

func newStyleListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all styles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStyleList()
		},
	}
}

func runStyleList() error {
	store := styles.NewDefaultStore()
	doc, err := store.Load()
	if err != nil {
		return err
	}

	names := store.List(doc)
	if len(names) == 0 {
		fmt.Println("(no styles)")
		return nil
	}

	for _, name := range names {
		entry := doc.Styles[name]
		mark := " "
		if store.IsActive(doc, name) {
			mark = "*"
		}
		badge := ""
		if len(entry.Refs) > 0 {
			badge = fmt.Sprintf(" \U0001F4CE%d", len(entry.Refs))
		}
		preview := snippetPreview(entry.Snippet, 60)
		fmt.Printf("%s %s [%s]%s: %s\n", mark, name, entry.Kind, badge, preview)
	}

	// Footer
	if len(doc.Default) > 0 {
		fmt.Printf("\n* = active default (%s)\n", strings.Join(doc.Default, ", "))
	} else {
		fmt.Println("\n(no active default — pass --style NAME to apply one)")
	}
	return nil
}

func snippetPreview(snippet string, maxLen int) string {
	if snippet == "" {
		return ""
	}
	// Join words by space (collapse whitespace)
	fields := strings.Fields(snippet)
	joined := strings.Join(fields, " ")
	if len(joined) <= maxLen {
		return joined
	}
	return joined[:maxLen] + "\u2026"
}

func newStyleShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show NAME",
		Short: "Show details of a style",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStyleShow(args[0])
		},
	}
}

func runStyleShow(name string) error {
	store := styles.NewDefaultStore()
	doc, err := store.Load()
	if err != nil {
		return err
	}
	entry, err := store.Get(doc, name)
	if err != nil {
		return err
	}

	fmt.Printf("kind: %s\n", entry.Kind)
	if entry.Snippet != "" {
		fmt.Printf("snippet: %s\n", entry.Snippet)
	}
	if len(entry.Refs) > 0 {
		fmt.Printf("refs: %s\n", strings.Join(entry.Refs, ", "))
	}
	fmt.Printf("path: %s\n", store.AssetDir(name))
	return nil
}

func newStyleAddCmd() *cobra.Command {
	var (
		refs     []string
		kind     string
		fromLast bool
	)

	cmd := &cobra.Command{
		Use:   "add NAME [SNIPPET]",
		Short: "Add or overwrite a style",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			snippet := ""
			if len(args) > 1 {
				snippet = args[1]
			}
			return runStyleAdd(name, snippet, refs, kind, fromLast)
		},
	}

	cmd.Flags().StringArrayVar(&refs, "ref", nil, "reference image path (repeatable)")
	cmd.Flags().StringVar(&kind, "kind", "style", "style or character")
	cmd.Flags().BoolVar(&fromLast, "from-last", false, "use last generated image as ref")

	return cmd
}

func runStyleAdd(name, snippet string, refs []string, kindStr string, fromLast bool) error {
	if err := styles.ValidateName(name); err != nil {
		return fmt.Errorf("error: %v", err)
	}

	store := styles.NewDefaultStore()
	doc, err := store.Load()
	if err != nil {
		return err
	}

	// Resolve --from-last
	if fromLast {
		lastPath, err := styles.ResolveFromLast(store.Dir())
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}
		refs = append(refs, lastPath)
	}

	// Must have at least snippet or refs
	if snippet == "" && len(refs) == 0 {
		return fmt.Errorf("error: provide a snippet, --ref, or --from-last")
	}

	var k styles.Kind
	switch kindStr {
	case "style":
		k = styles.KindStyle
	case "character":
		k = styles.KindCharacter
	default:
		return fmt.Errorf("error: --kind must be 'style' or 'character'")
	}

	entry := styles.Entry{
		Kind:    k,
		Snippet: snippet,
		Refs:    []string{},
	}

	if err := store.Add(doc, name, entry); err != nil {
		return err
	}

	// Copy ref images
	for i, refPath := range refs {
		filename, err := store.CopyRefImage(name, refPath, i+1)
		if err != nil {
			return err
		}
		e := doc.Styles[name]
		e.Refs = append(e.Refs, filename)
		doc.Styles[name] = e
	}

	return store.Save(doc)
}

func newStyleAddRefCmd() *cobra.Command {
	var fromLast bool

	cmd := &cobra.Command{
		Use:   "add-ref NAME [IMG...]",
		Short: "Add reference image(s) to an existing style",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			imgs := args[1:]
			return runStyleAddRef(name, imgs, fromLast)
		},
	}

	cmd.Flags().BoolVar(&fromLast, "from-last", false, "use last generated image as ref")
	return cmd
}

func runStyleAddRef(name string, imgs []string, fromLast bool) error {
	store := styles.NewDefaultStore()
	doc, err := store.Load()
	if err != nil {
		return err
	}

	if fromLast {
		lastPath, err := styles.ResolveFromLast(store.Dir())
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}
		imgs = append(imgs, lastPath)
	}

	if len(imgs) == 0 {
		return fmt.Errorf("error: provide at least one image or --from-last")
	}

	for _, img := range imgs {
		if _, err := store.AddRef(doc, name, img); err != nil {
			return err
		}
	}
	return store.Save(doc)
}

func newStyleRmRefCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm-ref NAME FILE",
		Short: "Remove a reference image from a style",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := styles.NewDefaultStore()
			doc, err := store.Load()
			if err != nil {
				return err
			}
			if err := store.RemoveRef(doc, args[0], args[1]); err != nil {
				return err
			}
			return store.Save(doc)
		},
	}
}

func newStyleRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm NAME",
		Short: "Remove a style",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := styles.NewDefaultStore()
			doc, err := store.Load()
			if err != nil {
				return err
			}
			wasDefault, err := store.Remove(doc, args[0])
			if err != nil {
				return err
			}
			if wasDefault {
				fmt.Fprintln(os.Stderr, "(was in the active default; removed from it)")
			}
			return store.Save(doc)
		},
	}
}

func newStyleUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use NAME [NAME...]",
		Short: "Set the active default style(s)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := styles.NewDefaultStore()
			doc, err := store.Load()
			if err != nil {
				return err
			}
			if err := store.Use(doc, args); err != nil {
				return err
			}
			return store.Save(doc)
		},
	}
}

func newStyleClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear the active default style set",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store := styles.NewDefaultStore()
			doc, err := store.Load()
			if err != nil {
				return err
			}
			store.Clear(doc)
			return store.Save(doc)
		},
	}
}

func newStyleResetCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset to built-in styles (destructive)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Fprint(os.Stderr, "This will delete all styles and assets. Continue? [y/N] ")
				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer != "y" && answer != "yes" {
					return fmt.Errorf("aborted")
				}
			}
			store := styles.NewDefaultStore()
			doc, err := store.Load()
			if err != nil {
				return err
			}
			store.Reset(doc)
			return store.Save(doc)
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation")
	return cmd
}
