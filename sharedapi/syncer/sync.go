package syncer

import (
	"context"
	"fmt"
	"sync"
)

var globalSyncer Syncer = &syncerImpl{
	registry: &registry{},
}
var globalSyncerMutex sync.Mutex

type Syncer interface {
	Sync(ctx context.Context) error
	Registry() Registry
}

func Get() Syncer {
	globalSyncerMutex.Lock()
	defer globalSyncerMutex.Unlock()
	return globalSyncer
}

func Set(s Syncer) Syncer {
	globalSyncerMutex.Lock()
	defer globalSyncerMutex.Unlock()
	prev := s
	globalSyncer = s
	return prev
}

type syncerImpl struct {
	registry Registry
}

func (s *syncerImpl) Registry() Registry {
	return s.registry
}

var _ Syncer = &syncerImpl{}

func (s *syncerImpl) Sync(ctx context.Context) error {
	var sr SyncRun
	for _, r := range s.Registry().Registered() {
		if err := r.Run(ctx, &sr); err != nil {
			return fmt.Errorf("error running %v: %w", r.Name(), err)
		}
	}
	return nil
}

func Sync() {
	ctx := context.Background()
	if err := Get().Sync(ctx); err != nil {
		fmt.Println("Error: ", err)
	}
}
