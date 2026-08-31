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
	root.AddCommand(newAddCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newUpgradeCmd())
	return root
}

func newAddCmd() *cobra.Command {
	var folder string
	var list bool
	cmd := &cobra.Command{
		Use:   "add [target-repo] <addon[,addon...]> [addon...]",
		Short: "List or install optional context add-ons",
		Long: `List the built-in add-on catalog or install one or more focused
documents into an existing layout-v2 scaffold. When the first positional
argument is a known add-on ID, all arguments are installed in the current
directory. Otherwise, with two or more arguments, the first is the target
repository. Prefix a repository path such as ./contracts if its name is also
an add-on ID. IDs may be separated by commas. Existing add-on outputs are never
overwritten; INDEX.md and config.json are updated transactionally. Default
target is the current directory.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if list {
				if len(args) != 0 {
					return fmt.Errorf("ctx add --list does not accept positional arguments")
				}
				for _, addon := range ListAddons() {
					description := addon.Description
					if addon.Default {
						description += " (default for new scaffolds)"
					}
					cmdPrintf(cmd, "%-10s %-24s %s\n", addon.ID, addon.Path, description)
				}
				return nil
			}
			if len(args) == 0 {
				return fmt.Errorf("specify at least one add-on ID, or use --list")
			}
			target, requested := splitAddCommandArgs(args)
			abs, err := filepath.Abs(target)
			if err != nil {
				return err
			}
			result, err := Add(abs, folder, requested)
			if err != nil {
				return err
			}
			cmdPrintf(cmd, "✓ installed add-on(s) %s in %s/%s\n", strings.Join(result.Addons, ", "), abs, folder)
			cmdPrintf(cmd, "  added %s and refreshed INDEX.md routing\n", strings.Join(result.Files, ", "))
			return nil
		},
	}
	cmd.Flags().StringVarP(&folder, "folder", "f", ".ctx", "context folder path to extend")
	cmd.Flags().BoolVar(&list, "list", false, "list available add-ons")
	return cmd
}

func splitAddCommandArgs(args []string) (string, []string) {
	if len(args) <= 1 {
		return ".", args
	}
	if first, err := normalizeAddonNames([]string{args[0]}); err == nil && len(first) > 0 {
		return ".", args
	}
	return args[0], args[1:]
}

// Execute runs the root command.
func Execute() error {
	return NewRootCmd().Execute()
}

func newInitCmd() *cobra.Command {
	var folder string
	var modeValue string
	var withAddons []string
	var withoutAddons []string
	cmd := &cobra.Command{
		Use:   "init [target-repo]",
		Short: "Initialize an agent-context folder in a target repo",
		Long: `Initialize a .ctx/ (or --folder) context folder from embedded templates.
The --folder value must be one top-level directory name containing only letters,
digits, '.', '_', or '-'; nested paths and spaces are not supported by init.
Team mode (default) leaves durable context visible to Git while local session
state remains ignored by the scaffold's .gitignore. Local mode excludes the
whole folder through .git/info/exclude. New scaffolds include the glossary
add-on by default; use --without glossary for the fixed core alone and --with
to select other add-ons. In a fresh clone of a team scaffold, init creates only
the missing ignored local continuation. ctx never stages or commits files.
Default target is the current directory.`,
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
			addons, err := resolveInitAddons(
				withAddons,
				withoutAddons,
				cmd.Flags().Changed("with") || cmd.Flags().Changed("without"),
			)
			if err != nil {
				return err
			}
			destExisted := false
			if info, statErr := os.Stat(filepath.Join(abs, folder)); statErr == nil && info.IsDir() {
				destExisted = true
			}
			if err := InitWithOptions(abs, InitOptions{Folder: folder, Mode: mode, Addons: addons}); err != nil {
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
	cmd.Flags().StringSliceVar(&withAddons, "with", nil, "add-on IDs to include with the defaults (repeat or separate with commas)")
	cmd.Flags().StringSliceVar(&withoutAddons, "without", nil, "default add-on IDs to omit (repeat or separate with commas)")
	return cmd
}

func resolveInitAddons(with, without []string, explicit bool) ([]string, error) {
	if !explicit {
		return nil, nil
	}
	withNames, err := normalizeAddonNames(with)
	if err != nil {
		return nil, err
	}
	withoutNames, err := normalizeAddonNames(without)
	if err != nil {
		return nil, err
	}
	withoutSet := make(map[string]bool, len(withoutNames))
	for _, id := range withoutNames {
		withoutSet[id] = true
	}
	defaultSet := make(map[string]bool)
	for _, id := range DefaultAddonIDs() {
		defaultSet[id] = true
	}
	for _, id := range withoutNames {
		if !defaultSet[id] {
			return nil, fmt.Errorf("add-on %q is not enabled by default; omit it from --with instead", id)
		}
	}
	for _, id := range withNames {
		if withoutSet[id] {
			return nil, fmt.Errorf("add-on %q cannot be passed to both --with and --without", id)
		}
	}

	selected := append(DefaultAddonIDs(), withNames...)
	selected, err = normalizeAddonNames(selected)
	if err != nil {
		return nil, err
	}
	resolved := make([]string, 0, len(selected))
	for _, id := range selected {
		if !withoutSet[id] {
			resolved = append(resolved, id)
		}
	}
	return resolved, nil
}

func newUpdateCmd() *cobra.Command {
	var folder string
	cmd := &cobra.Command{
		Use:   "update [target-repo]",
		Short: "Refresh managed blocks in a repo's context folder",
		Long: `Refresh the platform-managed documents declared by the scaffold's
persisted layout and add-ons, preserving all user content outside managed
blocks. Layout v2 matches strict named blocks and advances templateRevision;
layout v1 retains its frozen unnamed-marker and .ctx-version behavior. Default
target is the current directory.`,
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
				cmdPrintf(cmd, "✓ managed content is current; scaffold metadata refreshed\n")
			} else {
				cmdPrintf(cmd, "✓ refreshed %s and scaffold metadata\n", strings.Join(touched, ", "))
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
		Long: `Check that a target scaffold is healthy: config, layout/template
compatibility, mode-appropriate Git visibility, placeholders, managed markers,
and expected files. Exits non-zero if any check fails. Default target is the
current directory.`,
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

func newStatusCmd() *cobra.Command {
	var folder string
	cmd := &cobra.Command{
		Use:   "status [target-repo]",
		Short: "Check whether project context is current and ready",
		Long: `Check ctx:doc readiness metadata in every Markdown document below
<target>/<folder>/context/. Verified documents must name a real Git revision
and unchanged repository source paths. Size guidance is reported as a warning
and does not fail the command. This command does not modify the scaffold.
Default target is the current directory.`,
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
			report, err := Status(abs, folder)
			if err != nil {
				return err
			}
			failed := 0
			for _, check := range report.Checks {
				mark := "✓"
				switch check.State {
				case ContentNotReady:
					mark = "✗"
					failed++
				case ContentWarning:
					mark = "!"
				}
				if check.Detail == "" {
					cmdPrintf(cmd, "%s %s\n", mark, check.Path)
				} else {
					cmdPrintf(cmd, "%s %s — %s\n", mark, check.Path, check.Detail)
				}
			}
			if failed > 0 {
				return fmt.Errorf("%d content readiness check(s) failed", failed)
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
