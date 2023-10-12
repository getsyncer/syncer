package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/getsyncer/syncer-core/drift"

	"github.com/getsyncer/syncer-core/config/configloader"

	"github.com/getsyncer/syncer-core/config"

	"github.com/getsyncer/syncer-core/git"

	"github.com/cresta/pipe"
	"github.com/cresta/zapctx"
	"go.uber.org/zap"
)

type executeBase struct {
	git    git.Git
	loader configloader.ConfigLoader
	logger *zapctx.Logger
}

func newExecuteBase(git git.Git, loader configloader.ConfigLoader, logger *zapctx.Logger) *executeBase {
	return &executeBase{
		git:    git,
		loader: loader,
		logger: logger,
	}
}

func (r *executeBase) Execute(ctx context.Context, execCmd string, extraEnv string) (retErr error) {
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
	// Dynamic go build with tag "syncer"
	if err := pipe.NewPiped("go", "build", "-tags", "syncer", "-o", syncerBinaryPath, "sync.go").WithDir(wd).Run(ctx); err != nil {
		return fmt.Errorf("failed to build syncer: %w", err)
	}
	execEnv := envWithExtraParam(os.Environ(), "SYNCER_EXEC_CMD", execCmd)
	if extraEnv != "" {
		execEnv = append(execEnv, extraEnv)
	}
	if err := pipe.NewPiped(syncerBinaryPath).WithEnv(execEnv).Run(ctx); err != nil {
		return &failedToRunErr{
			root: err,
		}
	}
	return nil
}

type failedToRunErr struct {
	root error
}

func (f *failedToRunErr) Unwrap() error {
	return f.root
}

func (f *failedToRunErr) Error() string {
	return fmt.Sprintf("failed to run syncer: %v", f.root)
}

func envWithExtraParam(currentEnv []string, key string, value string) []string {
	var ret []string
	newVal := fmt.Sprintf("%s=%s", key, value)
	for idx, e := range currentEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[0] == key {
			ret = append(ret, currentEnv[:idx]...)
			ret = append(ret, newVal)
			ret = append(ret, currentEnv[idx+1:]...)
			return ret
		}
	}
	ret = append(ret, currentEnv...)
	ret = append(ret, newVal)
	return ret
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

func setupSync(ctx context.Context, logger *zapctx.Logger, rc *config.Root) (string, func() error, error) {
	// If there is a vendored sync file, do nothing
	vendoredFileLoc := filepath.Join(drift.DefaultSyncerDirectory, drift.DefaultSyncerMainFile)
	if _, err := os.Stat(vendoredFileLoc); err == nil {
		return drift.DefaultSyncerDirectory, func() error { return nil }, nil
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
	if err := initGoModAndImport(ctx, logger, rc, td, false); err != nil {
		return "", cleanup, fmt.Errorf("failed to setup syncer directory: %w", err)
	}
	return td, cleanup, nil
}

func initGoModAndImport(ctx context.Context, logger *zapctx.Logger, rc *config.Root, td string, assumeRootGoMod bool) error {
	logger.Debug(ctx, "Running go mod init")
	// 2. Dynamic `go mod init syncer` in that directory if there is no go.mod file
	// Note: Sometimes we assume the use of a root go mod file, and skip the mod init.  But we still need to run go get,
	//       since we need to add the tools files needed.
	if !assumeRootGoMod {
		if err := pipe.NewPiped("go", "mod", "init", "syncer").WithDir(td).Run(ctx); err != nil {
			return fmt.Errorf("failed to run go mod init")
		}
	}
	logger.Debug(ctx, "Creating syncer program")
	if err := generateSyncFile(ctx, logger, rc, filepath.Join(td, drift.DefaultSyncerMainFile)); err != nil {
		return fmt.Errorf("failed to generate syncer program: %w", err)
	}
	sourcesToGet := make([]string, 0, len(rc.Logic)+len(rc.Children))
	for _, l := range rc.Logic {
		sourcesToGet = append(sourcesToGet, l.Source)
	}
	for _, l := range rc.Children {
		sourcesToGet = append(sourcesToGet, l.Source)
	}
	for _, source := range sourcesToGet {
		logger.Debug(ctx, "Running go get", zap.String("package", source))
		if err := pipe.NewPiped("go", "get", source).WithDir(td).Run(ctx); err != nil {
			return fmt.Errorf("failed to run go get")
		}
	}
	logger.Debug(ctx, "Running go tidy")
	if err := pipe.NewPiped("go", "mod", "tidy").WithDir(td).Run(ctx); err != nil {
		return fmt.Errorf("failed to run go mod tidy")
	}
	return nil
}

func loadSyncerFile(ctx context.Context, g git.Git, loader configloader.ConfigLoader) (*config.Root, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	gr, err := g.FindGitRoot(ctx, wd)
	if err != nil {
		return nil, fmt.Errorf("failed to find git root: %w", err)
	}
	configFile, err := configloader.DefaultFindConfigFile(gr)
	if err != nil {
		return nil, fmt.Errorf("failed to find config file at default locations: %w", err)
	}
	if configFile == "" {
		return nil, fmt.Errorf("no config file found")
	}
	b, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	ret, err := loader.LoadConfig(ctx, bytes.NewReader(b))
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

func generateSyncFile(ctx context.Context, logger *zapctx.Logger, rc *config.Root, syncFilePath string) error {
	logger.Debug(ctx, "Creating syncer program")
	// 3. Create a syncer program there (sync.go)
	var syncerProg bytes.Buffer
	data := syncerTemplateData{
		AutogenMsg: drift.MagicTrackedString,
		Rc:         rc,
	}
	if err := syncerTemplate.Execute(&syncerProg, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	if err := os.WriteFile(syncFilePath, syncerProg.Bytes(), 0600); err != nil {
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

type syncerTemplateData struct {
	AutogenMsg string
	Rc         *config.Root
}

const defaultSyncerFile = `// Code generated by syncer vendor; DO NOT EDIT
//go:build syncer

// {{ .AutogenMsg }}

package main

import (
{{- range $val := .Rc.Logic }}
	_ "{{$val.SourceWithoutVersion}}"
{{- end }}
{{- range $val := .Rc.Children }}
	_ "{{$val.SourceWithoutVersion}}"
{{- end }}
	"github.com/getsyncer/syncer-core/syncerexec"
)

func main() {
	syncerexec.FromCli(syncerexec.DefaultFxOptions())
}
`
