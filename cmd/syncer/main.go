package main

import (
	"github.com/cresta/syncer/internal/cli"
	"github.com/cresta/syncer/internal/fxcli"
	"github.com/cresta/syncer/internal/git"
	"github.com/cresta/syncer/sharedapi/log"
	"github.com/cresta/syncer/sharedapi/syncer"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.WithLogger(log.NewFxLogger),
		cli.Module,
		log.Module,
		git.Module,
		syncer.Module,
		fxcli.Module,
		cli.ExecuteCliModule,
	).Run()
}
