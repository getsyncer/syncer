package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"go.uber.org/zap"

	"github.com/cresta/zapctx"

	"github.com/cresta/pipe"
	"github.com/cresta/syncer/internal/git"
	"github.com/cresta/syncer/sharedapi/syncer"
	"github.com/spf13/cobra"
)

type syncCmd struct {
	git    git.Git
	loader syncer.ConfigLoader
	logger *zapctx.Logger
}

func (r *syncCmd) MakeCobraCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Execute a sync, modifying any files that need to be modified",
		RunE:  r.RunE,
		Args:  cobra.NoArgs,
	}
}

func (r *syncCmd) RunE(cmd *cobra.Command, _ []string) (retErr error) {
	ctx := cmd.Context()
	// General steps:
	// 1. Find the syncer file
	r.logger.Debug(ctx, "Starting sync")
	cf, err := r.findSyncerFile(ctx)
	if err != nil {
		return fmt.Errorf("failed to find syncer file: %w", err)
	}
	// 2. Make a temp subdirectory
	td, err := os.MkdirTemp("", "syncer")
	if err != nil {
		return fmt.Errorf("failed to make temp dir: %w", err)
	}
	ctx = zapctx.With(ctx, zap.String("temp_dir", td))
	defer func() {
		if retErr == nil {
			if err := os.RemoveAll(td); err != nil {
				fmt.Printf("failed to remove temp dir: %v\n", err)
			}
		}
	}()
	r.logger.Debug(ctx, "Running go mod init")
	// 2. Run `go mod init syncer` in that directory if there is no go.mod file
	if err := pipe.NewPiped("go", "mod", "init", "syncer").WithDir(td).Run(ctx); err != nil {
		return fmt.Errorf("failed to run go mod init")
	}
	r.logger.Debug(ctx, "Loading config file")
	// Load the config file
	rc, err := r.loader.LoadConfig(ctx, cf)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	r.logger.Debug(ctx, "Creating syncer program")
	// 3. Create a syncer program there (sync.go)
	var syncerProg bytes.Buffer
	if err := syncerTemplate.Execute(&syncerProg, rc); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	syncFilePath := filepath.Join(td, "sync.go")
	if err := os.WriteFile(syncFilePath, syncerProg.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write syncer file: %w", err)
	}
	if r.logger.Unwrap(ctx).Level() <= zap.DebugLevel {
		r.logger.Debug(ctx, "Printing syncer file to stderr")
		if _, err := syncerProg.WriteTo(os.Stderr); err != nil {
			return fmt.Errorf("failed to write syncer file to stderr: %w", err)
		}
	}
	// 3a. Run "go get github.com/a/b" a bunch of times
	for _, l := range rc.Logic {
		r.logger.Debug(ctx, "Running go get", zap.String("package", l.Source))
		if err := pipe.NewPiped("go", "get", l.Source).WithDir(td).Run(ctx); err != nil {
			return fmt.Errorf("failed to run go get")
		}
	}
	r.logger.Debug(ctx, "Running go tidy")
	if err := pipe.NewPiped("go", "mod", "tidy").WithDir(td).Run(ctx); err != nil {
		return fmt.Errorf("failed to run go mod tidy")
	}
	// 4. Compile the syncer program (go build .)
	r.logger.Debug(ctx, "Running go build")
	if err := pipe.NewPiped("go", "build", "-o", "syncer", "sync.go").WithDir(td).Run(ctx); err != nil {
		return fmt.Errorf("failed to build syncer: %w", err)
	}
	// 5. Run it ( ./sync) inside this git repo's working directory
	syncerPath := filepath.Join(td, "syncer")
	r.logger.Debug(ctx, "Running syncer program", zap.String("path", syncerPath))
	if err := pipe.NewPiped(syncerPath).Run(ctx); err != nil {
		return fmt.Errorf("failed to run syncer: %w", err)
	}
	return err
}

func (r *syncCmd) findSyncerFile(ctx context.Context) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	gr, err := r.git.FindGitRoot(ctx, wd)
	if err != nil {
		return "", fmt.Errorf("failed to find git root: %w", err)
	}
	configFile, err := syncer.DefaultFindConfigFile(gr)
	if err != nil {
		return "", fmt.Errorf("failed to find config file at default locations: %w", err)
	}
	return configFile, nil
}

func newSyncCommand(logger *zapctx.Logger, git git.Git, loader syncer.ConfigLoader) *syncCmd {
	return &syncCmd{
		git:    git,
		loader: loader,
		logger: logger,
	}
}

var syncerTemplate = template.Must(template.New("syncer").Parse(defaultSyncerFile))

const defaultSyncerFile = `
package main

import (
{{ range $val := .Logic }}
     _ "{{$val.SourceWithoutVersion}}"
{{ end }}
	"github.com/cresta/syncer/sharedapi/syncer"
)

func main() {
	syncer.Sync(syncer.DefaultFxOptions())
}
`
