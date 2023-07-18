package syncer

import (
	"fmt"
	"sort"
	"sync"
)

type Registry interface {
	Register(r DriftSyncer) error
	Registered() []DriftSyncer
	Get(name string) (DriftSyncer, bool)
}

type registry struct {
	syncers []DriftSyncer
	mu      sync.Mutex
}

func NewRegistry(syncers []DriftSyncer) Registry {
	return &registry{
		syncers: syncers,
	}
}

func (r *registry) Get(name string) (DriftSyncer, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.syncers {
		if s.Name() == name {
			return s, true
		}
	}
	return nil, false
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

var _ Registry = &registry{}

type ErrSyncerAlreadyRegistered struct {
	Name string
}

func (e *ErrSyncerAlreadyRegistered) Error() string {
	return "syncer already registered: " + e.Name
}

func (r *registry) Register(s DriftSyncer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, r := range r.syncers {
		if r.Name() == s.Name() {
			return &ErrSyncerAlreadyRegistered{
				Name: s.Name(),
			}
		}
	}
	r.syncers = append(r.syncers, s)
	return nil
}

func (r *registry) Registered() []DriftSyncer {
	r.mu.Lock()
	defer r.mu.Unlock()
	sort.SliceStable(r.syncers, func(i, j int) bool {
		return r.syncers[i].Priority() < r.syncers[j].Priority() || r.syncers[i].Name() < r.syncers[j].Name()
	})
	return r.syncers
}
