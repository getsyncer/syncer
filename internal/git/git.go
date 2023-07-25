package git

import (
	"context"
	"fmt"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"os"
	"path/filepath"
)

type Git interface {
	FindGitRoot(ctx context.Context, loc string) (string, error)
	ListTrackedFiles(ctx context.Context, loc string) ([]string, error)
}

type gitOs struct{}

// ListTrackedFiles returns a list of files tracked by git.  It's similar to `git ls-files`
func (g *gitOs) ListTrackedFiles(ctx context.Context, loc string) ([]string, error) {
	rootLoc, err := g.FindGitRoot(ctx, loc)
	if err != nil {
		return nil, fmt.Errorf("failed to find git root: %w", err)
	}
	fs := osfs.New(rootLoc)
	if _, err := fs.Stat(git.GitDirName); err == nil {
		fs, err = fs.Chroot(git.GitDirName)
		if err != nil {
			return nil, fmt.Errorf("failed to chroot into git dir: %w", err)
		}
	}
	s := filesystem.NewStorageWithOptions(fs, cache.NewObjectLRUDefault(), filesystem.Options{KeepDescriptors: true})
	r, err := git.Open(s, fs)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repo: %w", err)
	}
	ref, err := r.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get head: %w", err)
	}
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}
	var ret []string
	err = tree.Files().ForEach(func(f *object.File) error {
		ret = append(ret, f.Name)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate files: %w", err)
	}
	return ret, nil
}

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
