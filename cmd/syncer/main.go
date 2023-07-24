package main

import (
	"github.com/getsyncer/syncer/internal/cli"
	"github.com/getsyncer/syncer/internal/git"
	"github.com/getsyncer/syncer/sharedapi/log"
	"github.com/getsyncer/syncer/sharedapi/syncer"
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
