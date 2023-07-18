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
