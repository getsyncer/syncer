package syncer

import (
	"context"
	"fmt"
	"sync"

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

type shortLivedSyncer struct {
	syncer Syncer
	sh     fx.Shutdowner
}

func newShortLivedSyncer(syncer Syncer, lc fx.Lifecycle, sh fx.Shutdowner) *shortLivedSyncer {
	ret := &shortLivedSyncer{
		sh:     sh,
		syncer: syncer,
	}

	lc.Append(fx.Hook{
		OnStart: ret.start,
		OnStop:  ret.stop,
	})

	return ret
}

func (s *shortLivedSyncer) start(_ context.Context) error {
	go s.run()
	return nil
}

func (s *shortLivedSyncer) stop(_ context.Context) error {
	return nil
}

func (s *shortLivedSyncer) run() {
	ctx := context.Background()
	if err := s.syncer.Sync(ctx); err != nil {
		fmt.Println("Error: ", err)
	}
	if err := s.sh.Shutdown(); err != nil {
		panic(err)
	}
}
