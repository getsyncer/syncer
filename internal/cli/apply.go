package cli

import (
	"github.com/spf13/cobra"
)

type applyCmd struct {
	executeBase *executeBase
}

func (r *applyCmd) MakeCobraCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "apply",
		Short: "Execute a sync, modifying any files that need to be modified",
		RunE:  r.RunE,
		Args:  cobra.NoArgs,
	}
}

func (r *applyCmd) RunE(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	return r.executeBase.Execute(ctx, "apply", "")
}

func newApplyCommand(execBase *executeBase) *applyCmd {
	return &applyCmd{
		executeBase: execBase,
	}
}
