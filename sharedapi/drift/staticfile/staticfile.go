package staticfile

import (
	"context"
	"fmt"

	"github.com/getsyncer/syncer/sharedapi/files"

	"github.com/getsyncer/syncer/sharedapi/syncer"
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

func (f *Syncer) Run(_ context.Context, runData *syncer.SyncRun) (*files.System[*files.StateWithChangeReason], error) {
	var cfg Config
	if err := runData.RunConfig.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal staticfile config: %w", err)
	}
	if cfg.Filename == "" {
		return nil, fmt.Errorf("filename is required")
	}
	var ret files.System[*files.StateWithChangeReason]
	path := cfg.Filename
	if err := ret.Add(files.Path(cfg.Filename), &files.StateWithChangeReason{
		ChangeReason: &files.ChangeReason{
			Reason: "staticfile",
		},
		State: files.State{
			Mode:          0644,
			Contents:      []byte(cfg.Content),
			FileExistence: files.FileExistencePresent,
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to add file %q: %w", path, err)
	}
	return &ret, nil
}

func (f *Syncer) Name() string {
	return "staticfile"
}

func (f *Syncer) Priority() syncer.Priority {
	return syncer.PriorityNormal
}

var _ syncer.DriftSyncer = &Syncer{}
