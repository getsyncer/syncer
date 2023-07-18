package syncer

import (
	"context"
	"fmt"
)

type ConfigMutator[T DriftConfig] interface {
	Mutate(ctx context.Context, runData *SyncRun, cfg T) (T, error)
}

type Mutatable[T DriftConfig] interface {
	AddMutator(mutator ConfigMutator[T])
}

type ConfigMutatorFunc[T DriftConfig] func(ctx context.Context, runData *SyncRun, cfg T) (T, error)

func (c ConfigMutatorFunc[T]) Mutate(ctx context.Context, runData *SyncRun, cfg T) (T, error) {
	return c(ctx, runData, cfg)
}

type MutatorList[T DriftConfig] struct {
	mutators []ConfigMutator[T]
}

func (m *MutatorList[T]) AddMutator(mutator ConfigMutator[T]) {
	m.mutators = append(m.mutators, mutator)
}

func (m *MutatorList[T]) Mutate(ctx context.Context, runData *SyncRun, cfg T) (T, error) {
	for _, mutator := range m.mutators {
		var err error
		cfg, err = mutator.Mutate(ctx, runData, cfg)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

func AddMutator[T DriftConfig](r Registry, name string, mutator ConfigMutator[T]) error {
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
	Name    string
}

func (s *SetupMutator[T]) Setup(_ context.Context, runData *SyncRun) error {
	if err := AddMutator[T](runData.Registry, s.Name, s.Mutator); err != nil {
		return fmt.Errorf("unable to add mutator: %w", err)
	}
	return nil
}
