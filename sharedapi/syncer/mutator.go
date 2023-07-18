package syncer

type ConfigMutator[T DriftConfig] interface {
	Mutate(T) (T, error)
}

type Mutatable[T DriftConfig] interface {
	AddMutator(mutator ConfigMutator[T])
}

type ConfigMutatorFunc[T DriftConfig] func(T) (T, error)

func (c ConfigMutatorFunc[T]) Mutate(cfg T) (T, error) {
	return c(cfg)
}

type MutatorList[T DriftConfig] struct {
	mutators []ConfigMutator[T]
}

func (m *MutatorList[T]) AddMutator(mutator ConfigMutator[T]) {
	m.mutators = append(m.mutators, mutator)
}

func (m *MutatorList[T]) Mutate(cfg T) (T, error) {
	for _, mutator := range m.mutators {
		var err error
		cfg, err = mutator.Mutate(cfg)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
