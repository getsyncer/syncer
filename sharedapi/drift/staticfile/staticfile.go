package staticfile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cresta/syncer/sharedapi/syncer"
)

type Config struct {
	// TODO: Figure out a way to support windows and unix paths
	Filename string
	Content  string
}

func New() *Syncer {
	return &Syncer{}
}

type Syncer struct {
}

func (f *Syncer) Run(_ context.Context, runData *syncer.SyncRun) error {
	var cfg Config
	if err := runData.RunConfig.Decode(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal staticfile config: %w", err)
	}
	if cfg.Filename == "" {
		return fmt.Errorf("filename is required")
	}
	// TODO: Make sub directories if required
	newPath := filepath.Join(runData.DestinationWorkingDir, cfg.Filename)
	if err := os.WriteFile(newPath, []byte(cfg.Content), 0644); err != nil {
		return fmt.Errorf("failed to write staticfile: %w", err)
	}
	return nil
}

func (f *Syncer) Name() string {
	return "staticfile"
}

func (f *Syncer) Priority() int {
	return syncer.PriorityNormal
}

var _ syncer.DriftSyncer = &Syncer{}
