package osfiles

import (
	"context"
	"fmt"

	"github.com/cresta/syncer/sharedapi/files"
)

type osLoader struct{}

func newOsLoader() *osLoader {
	return &osLoader{}
}

func (o *osLoader) ExecuteDiff(_ context.Context, path files.Path, d *files.Diff) error {
	if err := files.ExecuteDiffOnOs(path, d); err != nil {
		return fmt.Errorf("failed to execute diff for %s: %w", path, err)
	}
	return nil
}

func (o *osLoader) LoadState(_ context.Context, path files.Path) (*files.State, error) {
	state, err := files.NewStateFromPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load state for %s: %w", path, err)
	}
	return state, nil
}

var _ files.StateLoader = &osLoader{}
var _ files.DiffExecutor = &osLoader{}
