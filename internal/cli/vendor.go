package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/getsyncer/syncer-core/drift"

	"github.com/getsyncer/syncer-core/config/configloader"

	"github.com/cresta/zapctx"
	"github.com/getsyncer/syncer-core/git"
	"github.com/spf13/cobra"
)

type vendorCmd struct {
	git          git.Git
	loader       configloader.ConfigLoader
	logger       *zapctx.Logger
	useRootGoMod bool
}

func (r *vendorCmd) MakeCobraCommand() *cobra.Command {
	ret := &cobra.Command{
		Use:   "vendor",
		Short: "Execute a sync, modifying any files that need to be modified",
		RunE:  r.RunE,
		Args:  cobra.NoArgs,
	}
	ret.Flags().BoolVar(&r.useRootGoMod, "use-root-go-mod", true, "If true and a root go.mod exists, will use that instead of creating a new one")
	return ret
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
	fs, err := os.Stat(drift.DefaultSyncerMainFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat .syncer directory: %w", err)
		}
		if err := os.Mkdir(drift.DefaultSyncerMainFile, 0755); err != nil {
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
	if err := generateSyncFile(ctx, r.logger, rc, filepath.Join(drift.DefaultSyncerMainFile, drift.DefaultSyncerMainFile)); err != nil {
		return fmt.Errorf("failed to generate sync file: %w", err)
	}
	// 3. Look for a go.mod file inside the directory
	useExistingGoMod := r.useRootGoMod
	if useExistingGoMod {
		// If there is no root go mod, then we cannot use an existing one
		if _, err := os.Stat("go.mod"); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to stat go.mod file: %w", err)
			}
			useExistingGoMod = false
		}
	}
	// 4. If there is no go.mod file, run `go mod init syncer`
	if err := initGoModAndImport(ctx, r.logger, rc, drift.DefaultSyncerMainFile, useExistingGoMod); err != nil {
		return fmt.Errorf("failed to setup go.mod file: %w", err)
	}
	return nil
}

func newVendorCommand(logger *zapctx.Logger, git git.Git, loader configloader.ConfigLoader) *vendorCmd {
	return &vendorCmd{
		git:    git,
		loader: loader,
		logger: logger,
	}
}
