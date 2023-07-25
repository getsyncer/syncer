package syncer

import (
	"context"
	"fmt"

	"github.com/getsyncer/syncer/sharedapi/files"

	"github.com/getsyncer/syncer/sharedapi/files/existingfileparser"
)

type ConfigMutator[T DriftConfig] interface {
	Mutate(ctx context.Context, runData *SyncRun, loader files.StateLoader, cfg T) (T, error)
}

type Mutatable[T DriftConfig] interface {
	AddMutator(mutator ConfigMutator[T])
}

type ConfigMutatorFunc[T DriftConfig] func(ctx context.Context, runData *SyncRun, loader files.StateLoader, cfg T) (T, error)

func (c ConfigMutatorFunc[T]) Mutate(ctx context.Context, runData *SyncRun, loader files.StateLoader, cfg T) (T, error) {
	return c(ctx, runData, loader, cfg)
}

func SimpleConfigMutator[T DriftConfig](updateFunc func(T) (T, error)) ConfigMutator[T] {
	return ConfigMutatorFunc[T](func(_ context.Context, _ *SyncRun, _ files.StateLoader, cfg T) (T, error) {
		return updateFunc(cfg)
	})
}

type MutatorList[T DriftConfig] struct {
	mutators []ConfigMutator[T]
}

func (m *MutatorList[T]) AddMutator(mutator ConfigMutator[T]) {
	m.mutators = append(m.mutators, mutator)
}

func (m *MutatorList[T]) Mutate(ctx context.Context, runData *SyncRun, loader files.StateLoader, cfg T) (T, error) {
	for _, mutator := range m.mutators {
		var err error
		cfg, err = mutator.Mutate(ctx, runData, loader, cfg)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

func AddMutator[T DriftConfig](r Registry, name Name, mutator ConfigMutator[T]) error {
	s, ok := r.Get(name)
	if !ok {
		return fmt.Errorf("syncer %s not found", name)
	}
	asMutatable, ok := s.(Mutatable[T])
	if !ok {
		return fmt.Errorf("syncer %s is not mutatable", name)
	}
	asMutatable.AddMutator(mutator)
	return nil
}

type SetupMutator[T DriftConfig] struct {
	Mutator ConfigMutator[T]
	Name    Name
}

func (s *SetupMutator[T]) Setup(_ context.Context, runData *SyncRun) error {
	if err := AddMutator[T](runData.Registry, s.Name, s.Mutator); err != nil {
		return fmt.Errorf("unable to add mutator: %w", err)
	}
	return nil
}

type ParserApply[T DriftConfig] interface {
	ApplyFileParse(ctx context.Context, res *existingfileparser.ParseResult, path files.Path, runData *SyncRun, cfg T) (T, error)
}

type ParserApplyFunc[T DriftConfig] func(ctx context.Context, res *existingfileparser.ParseResult, path files.Path, runData *SyncRun, cfg T) (T, error)

func (p ParserApplyFunc[T]) ApplyFileParse(ctx context.Context, res *existingfileparser.ParseResult, path files.Path, runData *SyncRun, cfg T) (T, error) {
	return p(ctx, res, path, runData, cfg)
}

type ParserMutator[T DriftConfig] struct {
	Path  files.Path
	Conf  existingfileparser.ParseConfig
	Apply ParserApply[T]
}

func DefaultParseMutator[T ApplyableConfig[T]](path files.Path) *ParserMutator[T] {
	return &ParserMutator[T]{
		Path:  path,
		Conf:  existingfileparser.RecommendedNewlineSeparatedConfig(),
		Apply: ConfigApply[T](),
	}
}

func (p *ParserMutator[T]) Mutate(ctx context.Context, runData *SyncRun, loader files.StateLoader, cfg T) (T, error) {
	pr, err := existingfileparser.Parse(ctx, loader, p.Path, p.Conf)
	if err != nil {
		return cfg, fmt.Errorf("unable to parse: %w", err)
	}
	ret, err := p.Apply.ApplyFileParse(ctx, pr, p.Path, runData, cfg)
	if err != nil {
		return cfg, fmt.Errorf("unable to apply file parse: %w", err)
	}
	return ret, nil
}

var _ ConfigMutator[DriftConfig] = &ParserMutator[DriftConfig]{}

type ApplyableConfig[T DriftConfig] interface {
	ApplyParse(parse *existingfileparser.ParseResult) (T, error)
	DriftConfig
}

func ConfigApply[T ApplyableConfig[T]]() ParserApply[T] {
	return ParserApplyFunc[T](func(_ context.Context, res *existingfileparser.ParseResult, _ files.Path, _ *SyncRun, cfg T) (T, error) {
		ret, err := cfg.ApplyParse(res)
		if err != nil {
			return cfg, fmt.Errorf("unable to apply parse: %w", err)
		}
		return ret, nil
	})
}
