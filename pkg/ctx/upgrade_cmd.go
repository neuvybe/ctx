package ctx

import (
	"os"

	"github.com/spf13/cobra"
)

func newUpgradeCmd() *cobra.Command {
	var apiBase string
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade the ctx CLI to the latest release",
		Long: `Upgrade the ctx CLI to the latest GitHub release. For direct-binary installs
it self-replaces the running binary from the matching OS/arch release asset.
For package-manager installs (npm/brew/go install) it prints the right upgrade
command instead, since those managers own the binary.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			base := apiBase
			if base == "" {
				base = os.Getenv("CTX_RELEASES_API")
			}
			if base == "" {
				base = DefaultReleasesAPI
			}
			res, err := Upgrade(cmd.Context(), base)
			if err != nil {
				return err
			}
			if res.Message != "" {
				cmdPrintf(cmd, "%s\n", res.Message)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&apiBase, "api", "", "releases API base (default "+DefaultReleasesAPI+"; env CTX_RELEASES_API)")
	return cmd
}
