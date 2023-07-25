package files

import (
	"context"
	"fmt"
	"github.com/getsyncer/syncer/internal/git"
	"strings"
)

type Tracker interface {
	SyncedFiles(ctx context.Context, loc string, syncFlag string) (*System[*State], error)
}

type trackerImpl struct {
	git    git.Git
	loader StateLoader
}

func NewTracker(git git.Git, loader StateLoader) Tracker {
	return &trackerImpl{git: git, loader: loader}
}

func (t *trackerImpl) SyncedFiles(ctx context.Context, loc string, syncFlag string) (*System[*State], error) {
	files, err := t.git.ListTrackedFiles(ctx, loc)
	if err != nil {
		return nil, fmt.Errorf("failed to list git tracked files: %w", err)
	}
	var ret System[*State]
	for _, f := range files {
		s, err := t.loader.LoadState(ctx, Path(f))
		if err != nil {
			return nil, fmt.Errorf("failed to load state for %s: %w", f, err)
		}
		if strings.Contains(string(s.Contents), syncFlag) {
			if err := ret.Add(Path(f), s); err != nil {
				return nil, fmt.Errorf("failed to add state for %s: %w", f, err)
			}
		}
	}
	return &ret, nil
}

func ConvertToRemovals(s *System[*State]) (*System[*StateWithChangeReason], error) {
	var ret System[*StateWithChangeReason]
	for _, path := range s.Paths() {
		if err := ret.Add(path, &StateWithChangeReason{
			State: State{
				FileExistence: FileExistenceAbsent,
			},
			ChangeReason: &ChangeReason{
				Reason: "no-longer-tracked",
			},
		}); err != nil {
			return nil, fmt.Errorf("failed to add %s: %w", path, err)
		}
	}
	return &ret, nil
}
