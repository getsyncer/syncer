package syncer

import (
	"sort"
	"sync"
)

type Registry interface {
	Registered() []DriftSyncer
	Get(name string) (DriftSyncer, bool)
}

type registry struct {
	syncers []DriftSyncer
	mu      sync.Mutex
}

func NewRegistry(syncers []DriftSyncer) (Registry, error) {
	seen := map[string]struct{}{}
	for _, s := range syncers {
		if _, ok := seen[s.Name()]; ok {
			return nil, &ErrSyncerAlreadyRegistered{Name: s.Name()}
		}
		seen[s.Name()] = struct{}{}
	}
	return &registry{
		syncers: syncers,
	}, nil
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

var _ Registry = &registry{}

type ErrSyncerAlreadyRegistered struct {
	Name string
}

func (e *ErrSyncerAlreadyRegistered) Error() string {
	return "syncer already registered: " + e.Name
}

func (r *registry) Registered() []DriftSyncer {
	r.mu.Lock()
	defer r.mu.Unlock()
	sort.SliceStable(r.syncers, func(i, j int) bool {
		return r.syncers[i].Priority() < r.syncers[j].Priority() || r.syncers[i].Name() < r.syncers[j].Name()
	})
	return r.syncers
}
