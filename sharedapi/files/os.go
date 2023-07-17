package files

import (
	"context"
	"fmt"
)

type osLoader struct{}

func (o *osLoader) ExecuteDiff(_ context.Context, path Path, d *Diff) error {
	if err := ExecuteDiffOnOs(path, d); err != nil {
		return fmt.Errorf("failed to execute diff for %s: %w", path, err)
	}
	return nil
}

func (o *osLoader) LoadState(_ context.Context, path Path) (*State, error) {
	state, err := NewStateFromPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load state for %s: %w", path, err)
	}
	return state, nil
}

var _ StateLoader = &osLoader{}
var _ DiffExecutor = &osLoader{}
