package filetemplate

import (
	"context"

	"github.com/cresta/syncer/sharedapi/syncer"
)

type Config struct {
}

type SingleFileSync struct {
	SourceFilename string
	DestFilename   string
}

type Syncer struct {
}

func (f *Syncer) Run(ctx context.Context, runData *syncer.SyncRun) error {
	//TODO implement me
	panic("implement me")
}

func (f *Syncer) Name() string {
	return "filetemplate"
}

func (f *Syncer) Priority() int {
	return syncer.PriorityNormal
}

var _ syncer.DriftSyncer = &Syncer{}
