package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type syncCmd struct {
}

func (r *syncCmd) MakeCobraCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Execute a sync, modifying any files that need to be modified",
		RunE:  r.RunE,
		Args:  cobra.NoArgs,
	}
}

func (r *syncCmd) RunE(cmd *cobra.Command, _ []string) error {
	// General steps:
	// 1. Find the syncer file
	// 2. Make a subdirectory
	// 3. Create a syncer program there
	// 4. Compile the syncer program
	// 5. Run it
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Hello, world!\n")
	return err
}

func generateSyncCommand() *cobra.Command {
	r := &syncCmd{}
	return r.MakeCobraCommand()
}
