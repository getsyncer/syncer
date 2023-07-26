package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/getsyncer/syncer-core/git"

	"github.com/cresta/zapctx"
	"github.com/spf13/cobra"
)

type unvendorCmd struct {
	git    git.Git
	logger *zapctx.Logger
}

func (r *unvendorCmd) MakeCobraCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unvendor",
		Short: "Remove existing vendored files, if they exist",
		RunE:  r.RunE,
		Args:  cobra.NoArgs,
	}
}

func (r *unvendorCmd) RunE(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	changeBack, err := changeToGitRoot(ctx, r.git)
	if err != nil {
		return fmt.Errorf("failed to change to git root: %w", err)
	}
	defer func() {
		if err := changeBack(); err != nil {
			fmt.Printf("failed to change back to original directory: %v\n", err)
		}
	}()
	// Check for sync.go file
	exists, err := removeIfExist(filepath.Join(".syncer", "sync.go"))
	if err != nil {
		return fmt.Errorf("failed to remove sync.go: %w", err)
	}
	if !exists {
		return nil
	}
	_, err = removeIfExist(filepath.Join(".syncer", "go.mod"))
	if err != nil {
		return fmt.Errorf("failed to remove go.mod: %w", err)
	}
	_, err = removeIfExist(filepath.Join(".syncer", "go.sum"))
	if err != nil {
		return fmt.Errorf("failed to remove go.sum: %w", err)
	}
	return nil
}

func removeIfExist(path string) (bool, error) {
	if fs, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return false, fmt.Errorf("failed to stat %v: %w", path, err)
		}
		return false, nil
	} else if fs.IsDir() {
		return false, fmt.Errorf("%v is a directory", path)
	}
	if err := os.Remove(path); err != nil {
		return false, fmt.Errorf("failed to remove %v: %w", path, err)
	}
	return true, nil
}

func newUnvendorCmd(logger *zapctx.Logger, git git.Git) *unvendorCmd {
	return &unvendorCmd{
		git:    git,
		logger: logger,
	}
}
