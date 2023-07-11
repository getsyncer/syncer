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
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Hello, world!\n")
	return err
}

func generateSyncCommand() *cobra.Command {
	r := &syncCmd{}
	return r.MakeCobraCommand()
}
