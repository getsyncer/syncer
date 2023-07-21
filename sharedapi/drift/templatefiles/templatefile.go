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

type TemplateData[T TemplateConfig] struct {
	RunData *syncer.SyncRun
	Config  T
}

func NewGenerator[T TemplateConfig](files map[string]string, name string, priority syncer.Priority, decoder Decoder[T], logger *zapctx.Logger, setupLogic syncer.SetupSyncer, loader files.StateLoader) (*Generator[T], error) {
	if name == "" {
		return nil, fmt.Errorf("name must be set")
	}
	generatedTemplates := make(map[string]*template.Template, len(files))
	for k, v := range files {
		tmpl, err := NewTemplate(k, v)
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
		loader:     loader,
	}, nil
}

func NewTemplate(name string, data string) (*template.Template, error) {
	return template.New(name).Funcs(sprig.TxtFuncMap()).Parse(data)
}

type Decoder[T TemplateConfig] func(syncer.RunConfig) (T, error)

type NewModuleConfig[T TemplateConfig] struct {
	Name     string
	Files    map[string]string
	Priority syncer.Priority
	Decoder  Decoder[T]
	Setup    syncer.SetupSyncer
}

func NewModule[T TemplateConfig](config NewModuleConfig[T]) fx.Option {
	constructor := func(logger *zapctx.Logger, loader files.StateLoader) (*Generator[T], error) {
		return NewGenerator(config.Files, config.Name, config.Priority, config.Decoder, logger, config.Setup, loader)
	}
	return fx.Module(config.Name,
		fx.Provide(
			fx.Annotate(
				constructor,
				fx.As(new(syncer.DriftSyncer)),
				fx.ResultTags(`group:"syncers"`),
			),
		),
	)
}

func DefaultDecoder[T TemplateConfig]() func(runConfig syncer.RunConfig) (T, error) {
	return func(runConfig syncer.RunConfig) (T, error) {
		var cfg T
		if err := runConfig.Decode(&cfg); err != nil {
			return cfg, err
		}
		return cfg, nil
	}
}

type TemplateConfig interface {
}

type MergableConfig interface {
	// Merge into this object the defaults (if not set inside this object)
	Merge(defaults MergableConfig)
}

type Generator[T TemplateConfig] struct {
	files      map[string]*template.Template
	name       string
	priority   syncer.Priority
	decoder    func(syncer.RunConfig) (T, error)
	mutators   syncer.MutatorList[T]
	setupLogic syncer.SetupSyncer
	logger     *zapctx.Logger
	loader     files.StateLoader
}

func (f *Generator[T]) Setup(ctx context.Context, runData *syncer.SyncRun) error {
	if f.setupLogic != nil {
		return f.setupLogic.Setup(ctx, runData)
	}
	return nil
}

func (f *Generator[T]) AddMutator(mutator syncer.ConfigMutator[T]) {
	f.mutators.AddMutator(mutator)
}

func (f *Generator[T]) Run(ctx context.Context, runData *syncer.SyncRun) (*files.System[*files.StateWithChangeReason], error) {
	f.logger.Debug(ctx, "running templatefile", zap.String("name", f.name))
	cfg, err := f.decoder(runData.RunConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}
	cfg, err = f.mutators.Mutate(ctx, runData, f.loader, cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to mutate config: %w", err)
	}
	var ret files.System[*files.StateWithChangeReason]
	for k, v := range f.files {
		f.logger.Debug(ctx, "generating template", zap.String("destination", k))
		var err error
		var fileContent string
		if fileContent, err = f.generate(ctx, runData, cfg, v); err != nil {
			return nil, fmt.Errorf("unable to generate template for %s: %w", k, err)
		}
		if err := ret.Add(files.Path(k), &files.StateWithChangeReason{
			State: files.State{
				Contents:      []byte(fileContent),
				Mode:          0644,
				FileExistence: files.FileExistencePresent,
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

func (f *Generator[T]) generate(ctx context.Context, runData *syncer.SyncRun, config T, tmpl *template.Template) (string, error) {
	f.logger.Debug(ctx, "generating template", zap.Any("config", config))
	return ExecuteTemplateOnConfig(ctx, runData, config, tmpl)
}

func ExecuteTemplateOnConfig[T TemplateConfig](_ context.Context, runData *syncer.SyncRun, config T, tmpl *template.Template) (string, error) {
	d := TemplateData[T]{
		RunData: runData,
		Config:  config,
	}
	var into bytes.Buffer
	if err := tmpl.Execute(&into, d); err != nil {
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

var _ syncer.DriftSyncer = &Generator[TemplateConfig]{}
var _ syncer.SetupSyncer = &Generator[TemplateConfig]{}

type GenericConfigMutator[T TemplateConfig] struct {
	Name        string
	TemplateStr string
	MutateFunc  func(ctx context.Context, renderedTemplate string, cfg T) (T, error)
}

func (g *GenericConfigMutator[T]) Mutate(ctx context.Context, runData *syncer.SyncRun, _ files.StateLoader, cfg T) (T, error) {
	updatedBuildGoLib, err := NewTemplate(g.Name, g.TemplateStr)
	if err != nil {
		return cfg, fmt.Errorf("unable to parse template: %w", err)
	}
	res, err := ExecuteTemplateOnConfig(ctx, runData, cfg, updatedBuildGoLib)
	if err != nil {
		return cfg, fmt.Errorf("unable to execute template: %w", err)
	}
	return g.MutateFunc(ctx, res, cfg)
}
