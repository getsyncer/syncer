package files

import (
	"context"
	"fmt"
)

type StateLoader interface {
	LoadState(ctx context.Context, path Path) (*State, error)
}

func LoadAllState(ctx context.Context, paths []Path, loader StateLoader) (*System[*State], error) {
	var ret System[*State]
	for _, path := range paths {
		state, err := loader.LoadState(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("failed to load state for %s: %w", path, err)
		}
		if err := ret.Add(path, state); err != nil {
			return nil, fmt.Errorf("failed to add state for %s: %w", path, err)
		}
	}
	return &ret, nil
}
