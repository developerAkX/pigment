package cli

import (
	"fmt"

	"github.com/developerAkX/pigment/skills"
	"github.com/spf13/cobra"
)

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage pigment agent skills",
	}

	cmd.AddCommand(newSkillListCmd())
	cmd.AddCommand(newSkillInstallCmd())

	return cmd
}

func newSkillListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List embedded agent skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			skills, err := skills.List()
			if err != nil {
				return err
			}
			for _, s := range skills {
				fmt.Println(s.Name)
			}
			return nil
		},
	}
}

func newSkillInstallCmd() *cobra.Command {
	var (
		target string
		dir    string
		force  bool
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install agent skills to a skill directory",
		Long: `Install pigment's embedded agent skill to an agent skill
directory. Legacy split skills (pigment-generate, pigment-edit,
pigment-style) installed by older versions are removed automatically.

Targets:
  opencode  ~/.config/opencode/skills/ (default)
  claude    ~/.claude/skills/
  agents    ~/.agents/skills/

Use --dir to override the target directory.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			installed, err := skills.Install(target, dir, force)
			if err != nil {
				return err
			}
			for _, path := range installed {
				fmt.Printf("  installed: %s\n", path)
			}
			fmt.Printf("\n%d skill(s) installed.\n", len(installed))
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "opencode", "target platform (opencode, claude, agents)")
	cmd.Flags().StringVar(&dir, "dir", "", "override target directory")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite files not installed by pigment")

	return cmd
}
