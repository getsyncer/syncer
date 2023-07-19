package cli

import (
	"os"

	"github.com/spf13/cobra"
)

type planCmd struct {
	executeBase *executeBase
	exitCode    bool
}

func (r *planCmd) MakeCobraCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Plan a sync: printing any modifications that need to happen",
		RunE:  r.RunE,
		Args:  cobra.NoArgs,
	}
	cmd.PersistentFlags().BoolVarP(&r.exitCode, "exit-code", "e", false, "Exit with code 1 if there are changes")
	return cmd
}

func (r *planCmd) RunE(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	var extraEnv string
	if r.exitCode {
		extraEnv = "SYNCER_EXIT_CODE_ON_DIFF=true"
	}
	err := r.executeBase.Execute(ctx, "plan", extraEnv)
	if err != nil {
		if r.exitCode {
			os.Exit(1)
		}
		return err
	}
	return nil
}

func newPlanCommand(execBase *executeBase) *planCmd {
	return &planCmd{
		executeBase: execBase,
	}
}
