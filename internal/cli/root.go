package cli

import (
	"github.com/spf13/cobra"
)

type rootCmd struct {
}

func (r *rootCmd) MakeCobraCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "syncer",
		Short: "syncer is a tool for synchronizing files with template repositories",
	}
}

func WireRootCommand() *cobra.Command {
	r := &rootCmd{}
	ret := r.MakeCobraCommand()
	ret.AddCommand(generateSyncCommand())
	return ret
}
