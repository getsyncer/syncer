package syncer

import "context"

type Priority int

const (
	PriorityLowest  = 100
	PriorityLow     = 200
	PriorityNormal  = 300
	PriorityHigh    = 400
	PriorityHighest = 500
)

type DriftSyncer interface {
	Run(ctx context.Context, runData *SyncRun) error
	Name() string
	Priority() int
}
