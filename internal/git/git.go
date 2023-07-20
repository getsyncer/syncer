package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type Git interface {
	FindGitRoot(ctx context.Context, loc string) (string, error)
}

type gitOs struct{}

func NewGitOs() Git {
	return &gitOs{}
}

func (g *gitOs) FindGitRoot(_ context.Context, loc string) (string, error) {
	for i := 0; i < 500; i++ {
		gitDir := filepath.Join(loc, ".git")
		if fs, err := os.Stat(gitDir); err == nil && fs.IsDir() {
			return loc, nil
		}
		if loc == "/" || loc == "." || loc == "" {
			return "", nil
		}
		newLoc := filepath.Dir(loc)
		if newLoc == loc {
			return "", nil
		}
	}
	return "", fmt.Errorf("too many directories traversed")
}

var _ Git = &gitOs{}
