package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cresta/syncer/internal/git"
	"github.com/cresta/syncer/sharedapi/syncer"
	"github.com/cresta/zapctx"
	"github.com/spf13/cobra"
)

type vendorCmd struct {
	git    git.Git
	loader syncer.ConfigLoader
	logger *zapctx.Logger
}

func (r *vendorCmd) MakeCobraCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "vendor",
		Short: "Execute a sync, modifying any files that need to be modified",
		RunE:  r.RunE,
		Args:  cobra.NoArgs,
	}
}

func (r *vendorCmd) RunE(cmd *cobra.Command, _ []string) (retErr error) {
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
	// General steps:
	// 0. Go to git root directory
	// 1. Make .syncer directory
	fs, err := os.Stat(".syncer")
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat .syncer directory: %w", err)
		}
		if err := os.Mkdir(".syncer", 0755); err != nil {
			return fmt.Errorf("failed to make .syncer directory: %w", err)
		}
	} else if !fs.IsDir() {
		return fmt.Errorf(".syncer exists but is not a directory")
	}
	// 2. Find the syncer file
	rc, err := loadSyncerFile(ctx, r.git, r.loader)
	if err != nil {
		return fmt.Errorf("failed to find syncer file: %w", err)
	}
	// 2. Generate sync.go file with correct imports
	if err := generateSyncFile(ctx, r.logger, rc, filepath.Join(".syncer", "sync.go")); err != nil {
		return fmt.Errorf("failed to generate sync file: %w", err)
	}
	// 3. Look for a go.mod file inside the directory
	goModFileLoc := filepath.Join(".syncer", "go.mod")
	_, err = os.Stat(goModFileLoc)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat go.mod file: %w", err)
		}
		// 4. If there is no go.mod file, run `go mod init syncer`
		if err := initGoModAndImport(ctx, r.logger, rc, ".syncer"); err != nil {
			return fmt.Errorf("failed to setup go.mod file: %w", err)
		}
	}
	return nil
}

func newVendorCommand(logger *zapctx.Logger, git git.Git, loader syncer.ConfigLoader) *vendorCmd {
	return &vendorCmd{
		git:    git,
		loader: loader,
		logger: logger,
	}
}
