package templatefiles

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/cresta/syncer/sharedapi/files"

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

func NewGenerator[T any](files map[string]string, name string, priority syncer.Priority, decoder Decoder[T], logger *zapctx.Logger, setupLogic syncer.SetupSyncer) (*Generator[T], error) {
	generatedTemplates := make(map[string]*template.Template, len(files))
	for k, v := range files {
		tmpl, err := template.New(k).Funcs(sprig.TxtFuncMap()).Parse(v)
		if err != nil {
			return nil, fmt.Errorf("unable to parse template %q: %w", k, err)
		}
		generatedTemplates[k] = tmpl
	}
	return &Generator[T]{
		files:      generatedTemplates,
		name:       name,
		priority:   priority,
		decoder:    decoder,
		logger:     logger,
		setupLogic: setupLogic,
	}, nil
}

type Decoder[T any] func(syncer.RunConfig) (T, error)

func NewModule[T any](name string, files map[string]string, priority syncer.Priority, decoder Decoder[T], setupLogic syncer.SetupSyncer) fx.Option {
	constructor := func(logger *zapctx.Logger) (*Generator[T], error) {
		return NewGenerator(files, name, priority, decoder, logger, setupLogic)
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
	files      map[string]*template.Template
	name       string
	priority   syncer.Priority
	decoder    func(syncer.RunConfig) (T, error)
	mutators   []ConfigMutator[T]
	setupLogic syncer.SetupSyncer
	logger     *zapctx.Logger
}

func (f *Generator[T]) Setup(ctx context.Context, runData *syncer.SyncRun) error {
	if f.setupLogic != nil {
		return f.setupLogic.Setup(ctx, runData)
	}
	return nil
}

func (f *Generator[T]) AddMutator(mutator ConfigMutator[T]) {
	f.mutators = append(f.mutators, mutator)
}

func (f *Generator[T]) Run(ctx context.Context, runData *syncer.SyncRun) (*files.System[*files.StateWithChangeReason], error) {
	cfg, err := f.decoder(runData.RunConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}
	for _, v := range f.mutators {
		cfg = v(cfg)
	}
	var ret files.System[*files.StateWithChangeReason]
	for k, v := range f.files {
		var err error
		var fileContent string
		if fileContent, err = f.generate(ctx, runData, cfg, v, k); err != nil {
			return nil, fmt.Errorf("unable to generate template for %s: %w", k, err)
		}
		if err := ret.Add(files.Path(k), &files.StateWithChangeReason{
			State: files.State{
				Contents: []byte(fileContent),
				Mode:     0644,
			},
			ChangeReason: &files.ChangeReason{
				Reason: "template",
			},
		}); err != nil {
			return nil, fmt.Errorf("unable to add file %s: %w", k, err)
		}
	}
	return &ret, nil
}

func (f *Generator[T]) generate(ctx context.Context, runData *syncer.SyncRun, config T, tmpl *template.Template, destination string) (string, error) {
	f.logger.Debug(ctx, "generating template", zap.String("destination", destination), zap.Any("config", config))
	d := TemplateData[T]{
		RunData: runData,
		Config:  config,
	}
	var into bytes.Buffer
	if err := tmpl.Funcs(sprig.FuncMap()).Execute(&into, d); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return into.String(), nil
}

func (f *Generator[T]) Name() string {
	return f.name
}

func (f *Generator[T]) Priority() syncer.Priority {
	return f.priority
}

var _ syncer.DriftSyncer = &Generator[any]{}
var _ syncer.SetupSyncer = &Generator[any]{}
