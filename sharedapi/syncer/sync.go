package syncer

import (
	"context"
	"fmt"
	"os"

	"github.com/getsyncer/syncer/sharedapi/files/fileprinter/consoleprinter"

	"github.com/getsyncer/syncer/sharedapi/files/osfiles"

	"github.com/getsyncer/syncer/sharedapi/files"

	"github.com/getsyncer/syncer/sharedapi/log"

	"go.uber.org/fx"

	"go.uber.org/zap"

	"github.com/cresta/zapctx"
)

type Planner interface {
	Plan(ctx context.Context) (*files.System[*files.DiffWithChangeReason], error)
}

type Applier interface {
	Apply(ctx context.Context, stateDiff *files.System[*files.DiffWithChangeReason]) error
}

func NewPlanner(registry Registry, configLoader ConfigLoader, log *zapctx.Logger, stateLoader files.StateLoader) Planner {
	return &plannerImpl{
		registry:     registry,
		configLoader: configLoader,
		log:          log,
		stateLoader:  stateLoader,
	}
}

type plannerImpl struct {
	registry     Registry
	configLoader ConfigLoader
	log          *zapctx.Logger
	stateLoader  files.StateLoader
}

var _ Planner = &plannerImpl{}

func (s *plannerImpl) Plan(ctx context.Context) (*files.System[*files.DiffWithChangeReason], error) {
	s.log.Debug(ctx, "Starting plan")
	rc, err := ConfigFromFile(ctx, s.configLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	s.log.Debug(ctx, "config pre flatten", zap.Any("config", rc))
	if err := s.configLoader.FlattenChildren(ctx, rc); err != nil {
		return nil, fmt.Errorf("failed to flatten children: %w", err)
	}
	s.log.Debug(ctx, "Loaded config", zap.Any("config", rc))
	printConfigIfDebug(ctx, s.log, rc)
	if err := s.mergeConfigs(ctx, rc); err != nil {
		return nil, fmt.Errorf("failed to merge configs: %w", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	if err := s.loopAndExecute(ctx, rc, wd, loopAndExecuteSetup); err != nil {
		return nil, fmt.Errorf("failed to setup sync: %w", err)
	}
	changes := make([]*files.System[*files.StateWithChangeReason], 0, len(rc.Syncs))
	if err := s.loopAndExecute(ctx, rc, wd, loopAndExecuteRun(&changes)); err != nil {
		return nil, fmt.Errorf("failed to run sync: %w", err)
	}
	s.log.Debug(ctx, "Merging changes", zap.Any("changes", changes))
	finalExpectedState, err := files.SystemMerge(changes...)
	if err != nil {
		return nil, fmt.Errorf("failed to merge changes: %w", err)
	}
	allPaths := finalExpectedState.Paths()
	s.log.Debug(ctx, "Loading existing state", zap.Any("paths", allPaths))
	existingState, err := files.LoadAllState(ctx, allPaths, s.stateLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing state: %w", err)
	}
	s.log.Debug(ctx, "Calculating diff", zap.Any("existing", existingState), zap.Any("final", finalExpectedState))
	stateDiff, err := files.CalculateDiff(ctx, existingState, finalExpectedState)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate diff: %w", err)
	}
	return stateDiff, nil
}

func NewApplier(log *zapctx.Logger, diffExecutor files.DiffExecutor) Applier {
	return &applyImpl{
		log:          log,
		diffExecutor: diffExecutor,
	}
}

type applyImpl struct {
	log          *zapctx.Logger
	diffExecutor files.DiffExecutor
}

func (s *applyImpl) Apply(ctx context.Context, stateDiff *files.System[*files.DiffWithChangeReason]) error {
	s.log.Debug(ctx, "Executing diff", zap.Any("diff", stateDiff))
	if err := files.ExecuteAllDiffs(ctx, stateDiff, s.diffExecutor); err != nil {
		return fmt.Errorf("failed to execute diff: %w", err)
	}
	return nil
}

func printConfigIfDebug(ctx context.Context, logger *zapctx.Logger, rc *RootConfig) {
	if logger.Unwrap(ctx).Level() <= zap.DebugLevel {
		asY, err := rc.AsYaml()
		if err != nil {
			fmt.Printf("Failed to print config as yaml: %v\n", err)
		}
		fmt.Println(asY)
	}
}

type loopAndRunLogic func(ctx context.Context, syncer DriftSyncer, runData *SyncRun) error

func loopAndExecuteRun(changes *[]*files.System[*files.StateWithChangeReason]) func(ctx context.Context, syncer DriftSyncer, runData *SyncRun) error {
	return func(ctx context.Context, syncer DriftSyncer, runData *SyncRun) error {
		var runChanges *files.System[*files.StateWithChangeReason]
		var err error
		if runChanges, err = syncer.Run(ctx, runData); err != nil {
			return fmt.Errorf("error running %v: %w", syncer.Name(), err)
		}
		*changes = append(*changes, runChanges)
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

func (s *plannerImpl) mergeConfigs(ctx context.Context, rc *RootConfig) error {
	for idx := range rc.Syncs {
		r := rc.Syncs[idx]
		s.log.Debug(ctx, "Config before merge", zap.Any("config", r.Config), zap.String("config-as-yaml", ValOrErr(r.Config.AsYaml())))
		if err := rc.Syncs[idx].Config.Merge(rc.Config); err != nil {
			return fmt.Errorf("failed to merge run config: %w", err)
		}
		s.log.Debug(ctx, "Config after merge", zap.Any("config", r.Config), zap.String("config-as-yaml", ValOrErr(r.Config.AsYaml())))
	}
	return nil
}

func (s *plannerImpl) loopAndExecute(ctx context.Context, rc *RootConfig, wd string, toRun loopAndRunLogic) error {
	for _, r := range rc.Syncs {
		logic, exists := s.registry.Get(r.Logic)
		s.log.Debug(ctx, "config for this execute", zap.Any("run-config", r.Config))
		if !exists {
			return fmt.Errorf("logic %s not found", r.Logic)
		}
		sr := SyncRun{
			Registry:              s.registry,
			RootConfig:            rc,
			RunConfig:             r.Config,
			DestinationWorkingDir: wd,
		}
		if err := toRun(ctx, logic, &sr); err != nil {
			return fmt.Errorf("error running %v: %w", logic.Name(), err)
		}
	}
	return nil
}

func ValOrErr(v1 string, err error) string {
	if err != nil {
		return err.Error()
	}
	return v1
}

func DefaultFxOptions() fx.Option {
	return fx.Module("defaults",
		log.Module,
		Module,
	)
}

func FromCli(opts ...fx.Option) {
	var allOpts []fx.Option
	allOpts = append(allOpts, fx.WithLogger(log.NewFxLogger), osfiles.Module, consoleprinter.Module)
	allOpts = append(allOpts, opts...)
	allOpts = append(allOpts, globalFxRegistryInstance.Get()...)
	allOpts = append(allOpts, ExecuteCliModule)

	fx.New(allOpts...).Run()
}
