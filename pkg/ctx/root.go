package ctx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// NewRootCmd builds the root cobra command with --version and subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "ctx",
		Short: "ctx — a reusable agent-context platform",
		Long: `ctx scaffolds a .ctx/ agent-context folder into a repo and
keeps it healthy and upgraded. Team mode (the default) keeps durable project
context visible to Git while local session state remains ignored. Local mode
excludes the entire folder through Git's repo-local exclude file.`,
		SilenceUsage:  true,
		SilenceErrors: true,
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
	var modeValue string
	cmd := &cobra.Command{
		Use:   "init [target-repo]",
		Short: "Initialize an agent-context folder in a target repo",
		Long: `Initialize a .ctx/ (or --folder) context folder from embedded templates.
The --folder value must be one top-level directory name containing only letters,
digits, '.', '_', or '-'; nested paths and spaces are not supported by init.
Team mode (default) leaves durable context visible to Git while local session
state remains ignored by the scaffold's .gitignore. Local mode excludes the
whole folder through .git/info/exclude. In a fresh clone of a team scaffold,
init creates only the missing ignored local continuation. ctx never stages or
commits files. Default target is the current directory.`,
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
			mode, err := ParseMode(modeValue)
			if err != nil {
				return err
			}
			destExisted := false
			if info, statErr := os.Stat(filepath.Join(abs, folder)); statErr == nil && info.IsDir() {
				destExisted = true
			}
			if err := InitWithOptions(abs, InitOptions{Folder: folder, Mode: mode}); err != nil {
				return err
			}
			if destExisted {
				cmdPrintf(cmd, "✓ hydrated local state in %s/%s (ctx %s, mode %s)\n", abs, folder, Version, mode)
				cmdPrintf(cmd, "  durable context was left unchanged\n")
			} else {
				cmdPrintf(cmd, "✓ initialized %s/%s (ctx %s, mode %s)\n", abs, folder, Version, mode)
			}
			if mode == ModeTeam {
				cmdPrintf(cmd, "  durable context is visible to Git; %s/%s/local/ stays ignored\n", abs, folder)
				if !destExisted {
					cmdPrintf(cmd, "  review the generated files before staging or committing them\n")
				}
			} else {
				cmdPrintf(cmd, "  the entire %s/ folder is ignored through Git's repo-local exclude\n", folder)
			}
			cmdPrintf(cmd, "  next: point an agent at %s/%s/INDEX.md\n", abs, folder)
			return nil
		},
	}
	cmd.Flags().StringVarP(&folder, "folder", "f", ".ctx", "single top-level context folder name to create or hydrate")
	cmd.Flags().StringVar(&modeValue, "mode", string(ModeTeam), "visibility mode: team or local")
	return cmd
}

func newUpdateCmd() *cobra.Command {
	var folder string
	cmd := &cobra.Command{
		Use:   "update [target-repo]",
		Short: "Refresh managed blocks in a repo's context folder",
		Long: `Refresh the platform-managed blocks in <target>/<folder>/ (README.md, REVIEW.md)
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
				cmdPrintf(cmd, "✓ nothing to update (managed files absent or user-owned) — .ctx-version bumped to %s\n", Version)
			} else {
				cmdPrintf(cmd, "✓ refreshed %s — .ctx-version %s\n", strings.Join(touched, ", "), Version)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&folder, "folder", "f", ".ctx", "context folder path to update")
	return cmd
}

func newDoctorCmd() *cobra.Command {
	var folder string
	cmd := &cobra.Command{
		Use:   "doctor [target-repo]",
		Short: "Validate a repo's context-folder health",
		Long: `Check that a target scaffold is healthy: config/mode, version stamp,
mode-appropriate Git visibility, placeholders, managed markers, and expected
files. Exits non-zero if any check fails. Default target is the current
directory.`,
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
					cmdPrintf(cmd, "%s %s\n", mark, c.Name)
				} else {
					cmdPrintf(cmd, "%s %s — %s\n", mark, c.Name, c.Detail)
				}
			}
			if failed > 0 {
				return fmt.Errorf("%d doctor check(s) failed", failed)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&folder, "folder", "f", ".ctx", "context folder path to check")
	return cmd
}

func cmdPrintf(cmd *cobra.Command, format string, args ...any) {
	fmt.Fprintf(cmd.OutOrStdout(), format, args...)
}
