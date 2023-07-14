package templatefiles

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/cresta/syncer/sharedapi/syncer"
	"github.com/cresta/zapctx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type TemplateData[T any] struct {
	RunData *syncer.SyncRun
	Config  T
}

func NewGenerator[T any](files map[string]*template.Template, name string, priority int, decoder Decoder[T], logger *zapctx.Logger) *Generator[T] {
	return &Generator[T]{
		files:    files,
		name:     name,
		priority: priority,
		decoder:  decoder,
		logger:   logger,
	}
}

type Decoder[T any] func(syncer.RunConfig) (T, error)

func NewModule[T any](name string, files map[string]string, priority int, decoder Decoder[T]) fx.Option {
	tmpls := make(map[string]*template.Template)
	for k, v := range files {
		tmpls[k] = template.Must(template.New(k).Funcs(sprig.TxtFuncMap()).Parse(v))
	}
	constructor := func(logger *zapctx.Logger) *Generator[T] {
		return NewGenerator(tmpls, name, priority, decoder, logger)
	}
	return fx.Module(name,
		fx.Provide(
			fx.Annotate(
				constructor,
				fx.As(new(syncer.DriftSyncer)),
				fx.ResultTags(`group:"syncers"`),
			),
		),
	)
}

type ConfigMutator[T any] func(T) T

type Generator[T any] struct {
	files    map[string]*template.Template
	name     string
	priority int
	decoder  func(syncer.RunConfig) (T, error)
	mutators []ConfigMutator[T]
	logger   *zapctx.Logger
}

func (f *Generator[T]) AddMutator(mutator ConfigMutator[T]) {
	f.mutators = append(f.mutators, mutator)
}

func (f *Generator[T]) Run(ctx context.Context, runData *syncer.SyncRun) error {
	cfg, err := f.decoder(runData.RunConfig)
	if err != nil {
		return fmt.Errorf("unable to decode config: %w", err)
	}
	for _, v := range f.mutators {
		cfg = v(cfg)
	}
	for k, v := range f.files {
		if err := f.generate(ctx, runData, cfg, v, k); err != nil {
			return fmt.Errorf("unable to generate template for %s: %w", k, err)
		}
	}
	return nil
}

func (f *Generator[T]) generate(ctx context.Context, runData *syncer.SyncRun, config T, tmpl *template.Template, destination string) error {
	f.logger.Debug(ctx, "generating template", zap.String("destination", destination), zap.Any("config", config))
	pathDir := filepath.Dir(destination)
	if err := os.MkdirAll(pathDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", pathDir, err)
	}
	d := TemplateData[T]{
		RunData: runData,
		Config:  config,
	}
	var into bytes.Buffer
	if err := tmpl.Funcs(sprig.FuncMap()).Execute(&into, d); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	if err := os.WriteFile(destination, into.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", destination, err)
	}
	return nil
}

func (f *Generator[T]) Name() string {
	return f.name
}

func (f *Generator[T]) Priority() int {
	return f.priority
}

var _ syncer.DriftSyncer = &Generator[any]{}
