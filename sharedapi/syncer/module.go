package syncer

import (
	"context"
	"fmt"
	"sync"

	"github.com/cresta/syncer/internal/fxcli"
	"go.uber.org/fx"
)

var Module = fx.Module("syncer",
	fx.Provide(
		fx.Annotate(
			NewDefaultConfigLoader,
			fx.As(new(ConfigLoader)),
		),
		fx.Annotate(
			NewRegistry,
			fx.As(new(Registry)),
			fx.ParamTags(`group:"syncers"`),
		),
		fx.Annotate(
			NewSyncer,
			fx.As(new(Syncer)),
		),
	),
)

type globalFxRegistry struct {
	options []fx.Option
	mu      sync.Mutex
}

var globalFxRegistryInstance = &globalFxRegistry{}

func (g *globalFxRegistry) Register(opt fx.Option) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.options = append(g.options, opt)
}

func (g *globalFxRegistry) Get() []fx.Option {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.options
}

func FxRegister(opt fx.Option) {
	globalFxRegistryInstance.Register(opt)
}

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
	syncer Syncer
}

func NewFxCli(syncer Syncer) *FxCli {
	return &FxCli{syncer: syncer}
}

func (f *FxCli) Run() {
	ctx := context.Background()
	if err := f.syncer.Sync(ctx); err != nil {
		fmt.Println("Error: ", err)
	}
}
