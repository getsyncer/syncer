package templatefiles

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"

	"github.com/cresta/zapctx"
	"github.com/getsyncer/syncer/sharedapi/files"
	"github.com/getsyncer/syncer/sharedapi/syncer"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type TemplateData[T TemplateConfig] struct {
	RunData *syncer.SyncRun
	Config  T
}

func NewGenerator[T TemplateConfig](files map[string]string, name syncer.Name, priority syncer.Priority, decoder Decoder[T], logger *zapctx.Logger, setupLogic syncer.SetupSyncer, loader files.StateLoader, postGenProcessor PostGenProcessor) (*Generator[T], error) {
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
		files:            generatedTemplates,
		name:             name,
		priority:         priority,
		decoder:          decoder,
		logger:           logger,
		postGenProcessor: postGenProcessor,
		setupLogic:       setupLogic,
		loader:           loader,
	}, nil
}

func NewTemplate(name string, data string) (*template.Template, error) {
	tm := sprig.TxtFuncMap()
	tm["toYaml"] = toYAML
	return template.New(name).Funcs(tm).Parse(data)
}

func toYAML(v interface{}) (string, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("unable to marshal to yaml: %w", err)
	}
	return strings.TrimSuffix(string(data), "\n"), nil
}

// Taken from https://github.com/helm/helm/blob/main/pkg/engine/funcs.go#L30

type Decoder[T TemplateConfig] func(syncer.RunConfig) (T, error)

type NewModuleConfig[T TemplateConfig] struct {
	Name             syncer.Name
	Files            map[string]string
	Priority         syncer.Priority
	Decoder          Decoder[T]
	Setup            syncer.SetupSyncer
	PostGenProcessor PostGenProcessor
}

func NewModule[T TemplateConfig](config NewModuleConfig[T]) fx.Option {
	constructor := func(logger *zapctx.Logger, loader files.StateLoader) (*Generator[T], error) {
		if config.Priority == 0 {
			config.Priority = syncer.PriorityNormal
		}
		if config.Decoder == nil {
			config.Decoder = DefaultDecoder[T]()
		}
		if config.PostGenProcessor == nil {
			config.PostGenProcessor = PostGenProcessorList{}
		}
		return NewGenerator(config.Files, config.Name, config.Priority, config.Decoder, logger, config.Setup, loader, config.PostGenProcessor)
	}
	return fx.Module(config.Name.String(),
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
	files            map[string]*template.Template
	name             syncer.Name
	priority         syncer.Priority
	decoder          func(syncer.RunConfig) (T, error)
	mutators         syncer.MutatorList[T]
	setupLogic       syncer.SetupSyncer
	logger           *zapctx.Logger
	postGenProcessor PostGenProcessor
	loader           files.StateLoader
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

func (f *Generator[T]) PostGenProcess(ctx context.Context, fs *files.System[*files.StateWithChangeReason], runData *syncer.SyncRun) error {
	return f.postGenProcessor.PostGenProcess(ctx, fs, runData)
}

func (f *Generator[T]) Run(ctx context.Context, runData *syncer.SyncRun) (*files.System[*files.StateWithChangeReason], error) {
	f.logger.Debug(ctx, "running templatefile", zap.String("name", string(f.name)))
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
	if err := f.PostGenProcess(ctx, &ret, runData); err != nil {
		return nil, fmt.Errorf("unable to post gen process: %w", err)
	}
	return &ret, nil
}

type PostGenProcessor interface {
	PostGenProcess(ctx context.Context, fs *files.System[*files.StateWithChangeReason], runData *syncer.SyncRun) error
}

type PostGenProcessorList []PostGenProcessor

func (p PostGenProcessorList) PostGenProcess(ctx context.Context, fs *files.System[*files.StateWithChangeReason], runData *syncer.SyncRun) error {
	for _, v := range p {
		if err := v.PostGenProcess(ctx, fs, runData); err != nil {
			return fmt.Errorf("unable to post gen process: %w", err)
		}
	}
	return nil
}

type PostGenConfigMutator[T TemplateConfig] struct {
	ToMutate           syncer.Name
	TemplateName       string
	PostGenMutatorFunc PostGenMutatorFunc[T]
}

type PostGenMutatorFunc[T TemplateConfig] func(ctx context.Context, renderedTemplate string, cfg T) (T, error)

func (p *PostGenConfigMutator[T]) MakeMutator(renderedTemplate string) syncer.ConfigMutator[T] {
	return syncer.ConfigMutatorFunc[T](func(ctx context.Context, runData *syncer.SyncRun, loader files.StateLoader, cfg T) (T, error) {
		return p.PostGenMutatorFunc(ctx, renderedTemplate, cfg)
	})
}

func (p *PostGenConfigMutator[T]) PostGenProcess(_ context.Context, fs *files.System[*files.StateWithChangeReason], runData *syncer.SyncRun) error {
	s, exists := fs.Remove(files.Path(p.TemplateName))
	if !exists {
		return fmt.Errorf("unable to find template file %s", p.TemplateName)
	}
	if err := syncer.AddMutator(runData.Registry, p.ToMutate, p.MakeMutator(string(s.State.Contents))); err != nil {
		return fmt.Errorf("unable to add mutator: %w", err)
	}
	return nil
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

func (f *Generator[T]) Name() syncer.Name {
	return f.name
}

func (f *Generator[T]) Priority() syncer.Priority {
	return f.priority
}

var _ syncer.DriftSyncer = &Generator[TemplateConfig]{}
var _ syncer.SetupSyncer = &Generator[TemplateConfig]{}

type GenericConfigMutator[T TemplateConfig] struct {
	TemplateName string
	TemplateStr  string
	MutateFunc   func(ctx context.Context, renderedTemplate string, cfg T) (T, error)
}

func (g *GenericConfigMutator[T]) Mutate(ctx context.Context, runData *syncer.SyncRun, _ files.StateLoader, cfg T) (T, error) {
	updatedBuildGoLib, err := NewTemplate(g.TemplateName, g.TemplateStr)
	if err != nil {
		return cfg, fmt.Errorf("unable to parse template: %w", err)
	}
	res, err := ExecuteTemplateOnConfig(ctx, runData, cfg, updatedBuildGoLib)
	if err != nil {
		return cfg, fmt.Errorf("unable to execute template: %w", err)
	}
	return g.MutateFunc(ctx, res, cfg)
}
