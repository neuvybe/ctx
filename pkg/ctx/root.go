package ctx

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

// NewRootCmd builds the root cobra command with --version and subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "ctx",
		Short: "ctx — a reusable agent-context platform",
		Long: `ctx scaffolds a private .ctx/ agent-context folder into a repo and
(upcoming) keeps it upgraded. The folder is gitignored via .git/info/exclude
(repo-local, non-tracked) and holds a constitution-vs-log split of governing
files plus a reference-context schema for the project.`,
		SilenceUsage: true,
	}
	root.Version = Version
	root.SetVersionTemplate("ctx {{.Version}}\n")

	root.AddCommand(newInitCmd())
	// Upcoming: update, upgrade, doctor, review, status.
	return root
}

// Execute runs the root command.
func Execute() error {
	return NewRootCmd().Execute()
}

func newInitCmd() *cobra.Command {
	var folder string
	cmd := &cobra.Command{
		Use:   "init [target-repo]",
		Short: "Scaffold a .ctx/ context folder into a target repo",
		Long: `Scaffold a .ctx/ (or --folder) context folder into a target repo from the
embedded templates. Substitutes {{PROJECT}}/{{DATE}}, writes a .ctx-version
stamp, and adds the folder to the target's .git/info/exclude (repo-local,
non-tracked — never touches .gitignore). Default target is the current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) == 1 {
				target = args[0]
			}
			abs, err := filepath.Abs(target)
			if err != nil {
				return err
			}
			if err := Init(abs, folder); err != nil {
				return err
			}
			fmt.Printf("✓ scaffolded %s/%s (ctx %s)\n", abs, folder, Version)
			fmt.Printf("  next: point an agent at %s/%s/INDEX.md, or follow\n", abs, folder)
			fmt.Printf("  docs/fill-context-workflow.md to fill context/*.md\n")
			return nil
		},
	}
	cmd.Flags().StringVarP(&folder, "folder", "f", ".ctx", "folder name to create")
	return cmd
}