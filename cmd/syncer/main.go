package main

import (
	"github.com/getsyncer/syncer-core/git"
	"github.com/getsyncer/syncer-core/log"
	"github.com/getsyncer/syncer-core/syncer"
	"github.com/getsyncer/syncer/internal/cli"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.WithLogger(log.NewFxLogger),
		cli.Module,
		log.Module,
		git.Module,
		syncer.Module,
		cli.ExecuteCliModule,
	).Run()
}
