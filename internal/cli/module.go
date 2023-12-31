package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/getsyncer/syncer-core/fxcli"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

var Module = fx.Module("cli",
	fx.Provide(
		newRootCommand,
		newApplyCommand,
		newVendorCommand,
		newUnvendorCmd,
		newPlanCommand,
		newExecuteBase,
		newVersionCmd,
		RootCobraCommand,
	),
)

var ExecuteCliModule = fx.Module(
	"main",
	fx.Provide(
		fx.Annotate(
			NewFxCli,
			fx.As(new(fxcli.Main)),
		),
	),
)

type FxCli struct {
	cmd *cobra.Command
}

func NewFxCli(cmd *cobra.Command) *FxCli {
	return &FxCli{cmd: cmd}
}

func (f *FxCli) Run() {
	if err := f.cmd.ExecuteContext(context.Background()); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
