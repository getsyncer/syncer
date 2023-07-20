package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Note: These are injected at build time with the linker
	// I moved the defaults that goreleaser makes.
	// See https://goreleaser.com/cookbooks/using-main.version/

	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

type versionCmd struct {
}

func (r *versionCmd) MakeCobraCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE:  r.RunE,
		Args:  cobra.NoArgs,
	}
}

func (r *versionCmd) RunE(cmd *cobra.Command, _ []string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "syncer: version %s (Commit: %s Built: %s Builder: %s)\n", version, commit, date, builtBy)
	if err != nil {
		return fmt.Errorf("failed to write version to output stream: %w", err)
	}
	return nil
}

func newVersionCmd() *versionCmd {
	return &versionCmd{}
}
