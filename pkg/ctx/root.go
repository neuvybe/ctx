package ctx

import (
	"fmt"
	"path/filepath"
	"strings"

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
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newUpgradeCmd())
	// Upcoming: review, status.
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

func newUpdateCmd() *cobra.Command {
	var folder string
	cmd := &cobra.Command{
		Use:   "update [target-repo]",
		Short: "Refresh the managed blocks in a repo's .ctx/ from the installed CLI",
		Long: `Refresh the platform-managed blocks in <target>/.ctx/ (README.md, REVIEW.md)
from this CLI's embedded templates, preserving all user content (everything
outside <!-- ctx:managed --> blocks). Bumps .ctx-version. Files without markers
or missing files are skipped. Default target is the current directory.`,
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
			touched, err := Update(abs, folder)
			if err != nil {
				return err
			}
			if len(touched) == 0 {
				fmt.Printf("✓ nothing to update (managed files absent or user-owned) — .ctx-version bumped to %s\n", Version)
			} else {
				fmt.Printf("✓ refreshed %s — .ctx-version %s\n", strings.Join(touched, ", "), Version)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&folder, "folder", "f", ".ctx", "folder name to update")
	return cmd
}

func newDoctorCmd() *cobra.Command {
	var folder string
	cmd := &cobra.Command{
		Use:   "doctor [target-repo]",
		Short: "Validate a repo's .ctx/ health",
		Long: `Check that <target>/.ctx/ is healthy: folder exists, .ctx-version stamp
present, .git/info/exclude entry present, no leftover {{PROJECT}}/{{DATE}}
placeholders, managed markers balanced, expected files present. Exits non-zero
if any check fails. Default target is the current directory.`,
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
			checks, err := Doctor(abs, folder)
			if err != nil {
				return err
			}
			failed := 0
			for _, c := range checks {
				mark := "✓"
				if !c.OK {
					mark = "✗"
					failed++
				}
				if c.Detail == "" {
					fmt.Printf("%s %s\n", mark, c.Name)
				} else {
					fmt.Printf("%s %s — %s\n", mark, c.Name, c.Detail)
				}
			}
			if failed > 0 {
				return fmt.Errorf("%d doctor check(s) failed", failed)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&folder, "folder", "f", ".ctx", "folder name to check")
	return cmd
}