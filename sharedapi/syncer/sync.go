package syncer

import (
	"context"
	"fmt"
	"os"
	"sync"
)

var globalSyncer Syncer = &syncerImpl{
	registry: &registry{},
}
var globalSyncerMutex sync.Mutex

type Syncer interface {
	Sync(ctx context.Context) error
	Registry() Registry
	ConfigLoader() ConfigLoader
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
	registry     Registry
	configLoader DefaultConfigLoader
}

func (s *syncerImpl) ConfigLoader() ConfigLoader {
	return &s.configLoader
}

func (s *syncerImpl) Registry() Registry {
	return s.registry
}

var _ Syncer = &syncerImpl{}

func (s *syncerImpl) Sync(ctx context.Context) error {
	fmt.Println("A")
	rc, err := s.configLoader.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	fmt.Println("B")
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	fmt.Println("C")
	for _, r := range rc.Syncs {
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
		fmt.Println("D", logic.Name())
		if err := logic.Run(ctx, &sr); err != nil {
			return fmt.Errorf("error running %v: %w", logic.Name(), err)
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

func MustRegister(d DriftSyncer) {
	if err := Get().Registry().Register(d); err != nil {
		panic(err)
	}
}
