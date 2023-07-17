package syncer

import (
	"context"

	"github.com/cresta/syncer/sharedapi/files"
)

type Priority int

const (
	PriorityLowest  = 100
	PriorityLow     = 200
	PriorityNormal  = 300
	PriorityHigh    = 400
	PriorityHighest = 500
)

type DriftSyncer interface {
	Run(ctx context.Context, runData *SyncRun) (*files.System[*files.StateWithChangeReason], error)
	Name() string
	Priority() Priority
}

type SetupSyncer interface {
	Setup(ctx context.Context, runData *SyncRun) error
}

type SetupSyncerFunc func(ctx context.Context, runData *SyncRun) error

func (s SetupSyncerFunc) Setup(ctx context.Context, runData *SyncRun) error {
	return s(ctx, runData)
}
