package syncer

import (
	"context"
	"fmt"
	"os"

	"github.com/cresta/syncer/sharedapi/log"

	"go.uber.org/fx"

	"go.uber.org/zap"

	"github.com/cresta/zapctx"
)

type Syncer interface {
	Sync(ctx context.Context) error
	Registry() Registry
	ConfigLoader() ConfigLoader
}

func NewSyncer(registry Registry, configLoader ConfigLoader, log *zapctx.Logger) Syncer {
	return &syncerImpl{
		registry:     registry,
		configLoader: configLoader,
		log:          log,
	}
}

type syncerImpl struct {
	registry     Registry
	configLoader ConfigLoader
	log          *zapctx.Logger
}

func (s *syncerImpl) ConfigLoader() ConfigLoader {
	return s.configLoader
}

func (s *syncerImpl) Registry() Registry {
	return s.registry
}

var _ Syncer = &syncerImpl{}

func (s *syncerImpl) Sync(ctx context.Context) error {
	s.log.Info(ctx, "Starting sync")
	rc, err := s.configLoader.LoadConfig(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	for _, r := range rc.Syncs {
		s.log.Info(ctx, "Running sync", zap.String("logic", r.Logic))
		logic, exists := s.Registry().Get(r.Logic)
		if !exists {
			return fmt.Errorf("logic %s not found", r.Logic)
		}
		sr := SyncRun{
			Registry:              s.Registry(),
			RootConfig:            rc,
			RunConfig:             RunConfig{Node: r.Config},
			DestinationWorkingDir: wd,
		}
		if err := logic.Run(ctx, &sr); err != nil {
			return fmt.Errorf("error running %v: %w", logic.Name(), err)
		}
	}
	return nil
}

func DefaultFxOptions() fx.Option {
	return fx.Module("defaults",
		log.Module,
		Module,
	)
}

func Sync(opts ...fx.Option) {
	var allOpts []fx.Option
	allOpts = append(allOpts, fx.WithLogger(log.NewFxLogger))
	allOpts = append(allOpts, opts...)
	allOpts = append(allOpts, globalFxRegistryInstance.Get()...)
	allOpts = append(allOpts, ExecuteCliModule)

	fx.New(allOpts...).Run()
}
