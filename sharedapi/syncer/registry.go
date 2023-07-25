package syncer

import (
	"sort"
	"sync"
)

type Name string

func (n Name) String() string {
	return string(n)
}

type Registry interface {
	Registered() []DriftSyncer
	Get(name Name) (DriftSyncer, bool)
}

type registry struct {
	syncers []DriftSyncer
	mu      sync.Mutex
}

func NewRegistry(syncers []DriftSyncer) (Registry, error) {
	seen := map[string]struct{}{}
	for _, s := range syncers {
		if _, ok := seen[string(s.Name())]; ok {
			return nil, &ErrSyncerAlreadyRegistered{Name: string(s.Name())}
		}
		seen[string(s.Name())] = struct{}{}
	}
	return &registry{
		syncers: syncers,
	}, nil
}

func (r *registry) Get(name Name) (DriftSyncer, bool) {
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
