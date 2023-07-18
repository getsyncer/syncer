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

func newRootCommand() *rootCmd {
	return &rootCmd{}
}

func RootCobraCommand(r *rootCmd, s *syncCmd, v *vendorCmd) *cobra.Command {
	ret := r.MakeCobraCommand()
	ret.AddCommand(s.MakeCobraCommand())
	ret.AddCommand(v.MakeCobraCommand())
	return ret
}
