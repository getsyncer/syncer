package syncer

import (
	"context"
	"fmt"
	"os"

	"github.com/cresta/syncer/sharedapi/files/osfiles"

	"github.com/cresta/syncer/sharedapi/files"

	"github.com/cresta/syncer/sharedapi/log"

	"go.uber.org/fx"

	"go.uber.org/zap"

	"github.com/cresta/zapctx"
)

type Syncer interface {
	Apply(ctx context.Context) error
	Registry() Registry
	ConfigLoader() ConfigLoader
}

func NewSyncer(registry Registry, configLoader ConfigLoader, log *zapctx.Logger, stateLoader files.StateLoader, diffExecutor files.DiffExecutor) Syncer {
	return &syncerImpl{
		registry:     registry,
		configLoader: configLoader,
		log:          log,
		stateLoader:  stateLoader,
		diffExecutor: diffExecutor,
	}
}

type syncerImpl struct {
	registry     Registry
	configLoader ConfigLoader
	log          *zapctx.Logger
	stateLoader  files.StateLoader
	diffExecutor files.DiffExecutor
}

func (s *syncerImpl) ConfigLoader() ConfigLoader {
	return s.configLoader
}

func (s *syncerImpl) Registry() Registry {
	return s.registry
}

var _ Syncer = &syncerImpl{}

func (s *syncerImpl) Apply(ctx context.Context) error {
	s.log.Info(ctx, "Starting sync")
	rc, err := s.configLoader.LoadConfig(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	s.log.Debug(ctx, "Loaded config", zap.Any("config", rc))
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	if err := s.loopAndExecute(ctx, rc, wd, loopAndExecuteSetup); err != nil {
		return fmt.Errorf("failed to setup sync: %w", err)
	}
	changes := make([]*files.System[*files.StateWithChangeReason], 0, len(rc.Syncs))
	if err := s.loopAndExecute(ctx, rc, wd, loopAndExecuteRun(changes)); err != nil {
		return fmt.Errorf("failed to run sync: %w", err)
	}
	finalExpectedState, err := files.SystemMerge(changes...)
	if err != nil {
		return fmt.Errorf("failed to merge changes: %w", err)
	}
	allPaths := finalExpectedState.Paths()
	existingState, err := files.LoadAllState(ctx, allPaths, s.stateLoader)
	if err != nil {
		return fmt.Errorf("failed to load existing state: %w", err)
	}
	stateDiff, err := files.CalculateDiff(ctx, existingState, finalExpectedState)
	if err != nil {
		return fmt.Errorf("failed to calculate diff: %w", err)
	}
	if err := files.ExecuteAllDiffs(ctx, stateDiff, s.diffExecutor); err != nil {
		return fmt.Errorf("failed to execute diff: %w", err)
	}
	return nil
}

type loopAndRunLogic func(ctx context.Context, syncer DriftSyncer, runData *SyncRun) error

func loopAndExecuteRun(changes []*files.System[*files.StateWithChangeReason]) func(ctx context.Context, syncer DriftSyncer, runData *SyncRun) error {
	return func(ctx context.Context, syncer DriftSyncer, runData *SyncRun) error {
		var runChanges *files.System[*files.StateWithChangeReason]
		var err error
		if runChanges, err = syncer.Run(ctx, runData); err != nil {
			return fmt.Errorf("error running %v: %w", syncer.Name(), err)
		}
		changes = append(changes, runChanges)
		return nil
	}
}

func loopAndExecuteSetup(ctx context.Context, syncer DriftSyncer, runData *SyncRun) error {
	if canSetup, ok := syncer.(SetupSyncer); ok {
		if err := canSetup.Setup(ctx, runData); err != nil {
			return fmt.Errorf("error setting up %v: %w", syncer.Name(), err)
		}
	}
	return nil
}

func (s *syncerImpl) loopAndExecute(ctx context.Context, rc *RootConfig, wd string, toRun loopAndRunLogic) error {
	for _, r := range rc.Syncs {
		s.log.Debug(ctx, "Running sync", zap.String("logic", r.Logic), zap.Any("run-cfg", r.Config))
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
		if err := toRun(ctx, logic, &sr); err != nil {
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

func Apply(opts ...fx.Option) {
	var allOpts []fx.Option
	allOpts = append(allOpts, fx.WithLogger(log.NewFxLogger), osfiles.Module)
	allOpts = append(allOpts, opts...)
	allOpts = append(allOpts, globalFxRegistryInstance.Get()...)
	allOpts = append(allOpts, ExecuteCliModule)

	fx.New(allOpts...).Run()
}
