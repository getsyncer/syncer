package main

import (
	"github.com/getsyncer/syncer-core/log"
	"github.com/getsyncer/syncer-core/syncerexec"
	"github.com/getsyncer/syncer/internal/cli"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		syncerexec.DefaultFxOptions(),
		log.ModuleProd,
		cli.Module,
		cli.ExecuteCliModule,
	).Run()
}
