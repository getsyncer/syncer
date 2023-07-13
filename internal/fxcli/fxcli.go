package fxcli

import (
	"context"

	"go.uber.org/fx"
)

var Module = fx.Module("syncer",
	fx.Provide(
		NewFxCLI,
	),
	fx.Invoke(func(_ *FxCLI) {}),
)

type Main interface {
	Run()
}

type FxCLI struct {
	toRun Main
	sh    fx.Shutdowner
}

func NewFxCLI(toRun Main, lc fx.Lifecycle, sh fx.Shutdowner) *FxCLI {
	ret := &FxCLI{
		sh:    sh,
		toRun: toRun,
	}

	lc.Append(fx.Hook{
		OnStart: ret.start,
		OnStop:  ret.stop,
	})

	return ret
}

func (s *FxCLI) start(_ context.Context) error {
	go s.run()
	return nil
}

func (s *FxCLI) stop(_ context.Context) error {
	return nil
}

func (s *FxCLI) run() {
	defer func() {
		if err := s.sh.Shutdown(); err != nil {
			panic(err)
		}
	}()
	s.toRun.Run()
}
