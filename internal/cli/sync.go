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
	rc, err := loadSyncerFile(ctx, r.git, r.loader)
	if err != nil {
		return fmt.Errorf("failed to find syncer file: %w", err)
	}
	wd, cleanup, err := setupSync(ctx, r.logger, rc)
	if cleanup != nil {
		defer func() {
			if err := cleanup(); err != nil {
				r.logger.Warn(ctx, "failed to cleanup", zap.Error(err))
			}
		}()
	}
	if err != nil {
		return fmt.Errorf("failed to setup sync: %w", err)
	}

	// 4. Compile the syncer program (go build .)
	r.logger.Debug(ctx, "Running go build")
	syncerBinaryPath, err := tempPathForSyncer()
	if err != nil {
		return fmt.Errorf("failed to get temp path for syncer: %w", err)
	}
	r.logger.Debug(ctx, "Running syncer program", zap.String("path", syncerBinaryPath))
	// Run go build with tag "syncer"
	if err := pipe.NewPiped("go", "build", "-tags", "syncer", "-o", syncerBinaryPath, "sync.go").WithDir(wd).Run(ctx); err != nil {
		return fmt.Errorf("failed to build syncer: %w", err)
	}
	if err := pipe.NewPiped(syncerBinaryPath).Run(ctx); err != nil {
		return fmt.Errorf("failed to run syncer: %w", err)
	}
	return err
}

func tempPathForSyncer() (string, error) {
	f, err := os.CreateTemp("", "syncer")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	fileName := f.Name()
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}
	return fileName, nil
}

func setupSync(ctx context.Context, logger *zapctx.Logger, rc *syncer.RootConfig) (string, func() error, error) {
	// If there is a vendored sync file, do nothing
	vendoredFileLoc := filepath.Join(".syncer", "sync.go")
	if _, err := os.Stat(vendoredFileLoc); err == nil {
		return ".syncer", func() error { return nil }, nil
	}

	// 2. Make a temp subdirectory
	td, err := os.MkdirTemp("", "syncer")
	if err != nil {
		return "", nil, fmt.Errorf("failed to make temp dir: %w", err)
	}
	ctx = zapctx.With(ctx, zap.String("temp_dir", td))
	cleanup := func() error {
		return os.RemoveAll(td)
	}
	logger.Debug(ctx, "Running go mod init")
	if err := initGoModAndImport(ctx, logger, rc, td); err != nil {
		return "", cleanup, fmt.Errorf("failed to setup syncer directory: %w", err)
	}
	return td, cleanup, nil
}

func initGoModAndImport(ctx context.Context, logger *zapctx.Logger, rc *syncer.RootConfig, td string) error {
	logger.Debug(ctx, "Running go mod init")
	// 2. Run `go mod init syncer` in that directory if there is no go.mod file
	if err := pipe.NewPiped("go", "mod", "init", "syncer").WithDir(td).Run(ctx); err != nil {
		return fmt.Errorf("failed to run go mod init")
	}
	logger.Debug(ctx, "Creating syncer program")
	if err := generateSyncFile(ctx, logger, rc, filepath.Join(td, "sync.go")); err != nil {
		return fmt.Errorf("failed to generate syncer program: %w", err)
	}
	for _, l := range rc.Logic {
		logger.Debug(ctx, "Running go get", zap.String("package", l.Source))
		if err := pipe.NewPiped("go", "get", l.Source).WithDir(td).Run(ctx); err != nil {
			return fmt.Errorf("failed to run go get")
		}
	}
	logger.Debug(ctx, "Running go tidy")
	if err := pipe.NewPiped("go", "mod", "tidy").WithDir(td).Run(ctx); err != nil {
		return fmt.Errorf("failed to run go mod tidy")
	}
	return nil
}

func loadSyncerFile(ctx context.Context, g git.Git, loader syncer.ConfigLoader) (*syncer.RootConfig, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	gr, err := g.FindGitRoot(ctx, wd)
	if err != nil {
		return nil, fmt.Errorf("failed to find git root: %w", err)
	}
	configFile, err := syncer.DefaultFindConfigFile(gr)
	if err != nil {
		return nil, fmt.Errorf("failed to find config file at default locations: %w", err)
	}
	if configFile == "" {
		return nil, fmt.Errorf("no config file found")
	}
	ret, err := loader.LoadConfig(ctx, configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}
	return ret, nil
}

func changeToGitRoot(ctx context.Context, g git.Git) (func() error, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	gr, err := g.FindGitRoot(ctx, wd)
	if err != nil {
		return nil, fmt.Errorf("failed to find git root: %w", err)
	}
	if err := os.Chdir(gr); err != nil {
		return nil, fmt.Errorf("failed to change to git root: %w", err)
	}
	return func() error {
		return os.Chdir(wd)
	}, nil
}

func newSyncCommand(logger *zapctx.Logger, git git.Git, loader syncer.ConfigLoader) *syncCmd {
	return &syncCmd{
		git:    git,
		loader: loader,
		logger: logger,
	}
}

func generateSyncFile(ctx context.Context, logger *zapctx.Logger, rc *syncer.RootConfig, syncFilePath string) error {
	logger.Debug(ctx, "Creating syncer program")
	// 3. Create a syncer program there (sync.go)
	var syncerProg bytes.Buffer
	if err := syncerTemplate.Execute(&syncerProg, rc); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	if err := os.WriteFile(syncFilePath, syncerProg.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write syncer file: %w", err)
	}
	if logger.Unwrap(ctx).Level() <= zap.DebugLevel {
		logger.Debug(ctx, "Printing syncer file to stderr")
		if _, err := syncerProg.WriteTo(os.Stderr); err != nil {
			return fmt.Errorf("failed to write syncer file to stderr: %w", err)
		}
	}
	return nil
}

var syncerTemplate = template.Must(template.New("syncer").Parse(defaultSyncerFile))

const defaultSyncerFile = `//go:build syncer
// +build syncer

package main

import (
{{ range $val := .Logic }}
     _ "{{$val.SourceWithoutVersion}}"
{{- end }}
	"github.com/cresta/syncer/sharedapi/syncer"
)

func main() {
	syncer.Apply(syncer.DefaultFxOptions())
}
`
