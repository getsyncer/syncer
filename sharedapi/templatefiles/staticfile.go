package templatefiles

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"go.uber.org/zap"

	"github.com/cresta/zapctx"

	"github.com/Masterminds/sprig/v3"
	"github.com/cresta/syncer/sharedapi/syncer"
	"go.uber.org/fx"
)

type TemplateData struct {
	RunData *syncer.SyncRun
	Config  interface{}
}

func NewGenerator(files map[string]*template.Template, name string, priority int, decoder Decoder, logger *zapctx.Logger) *Generator {
	return &Generator{
		files:    files,
		name:     name,
		priority: priority,
		decoder:  decoder,
		logger:   logger,
	}
}

type Decoder func(syncer.RunConfig) (interface{}, error)

func NewModule(name string, files map[string]string, priority int, decoder Decoder) fx.Option {
	tmpls := make(map[string]*template.Template)
	for k, v := range files {
		tmpls[k] = template.Must(template.New(k).Funcs(sprig.TxtFuncMap()).Parse(v))
	}
	constructor := func(logger *zapctx.Logger) *Generator {
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

type Generator struct {
	files    map[string]*template.Template
	name     string
	priority int
	decoder  func(syncer.RunConfig) (interface{}, error)
	logger   *zapctx.Logger
}

func (f *Generator) Run(ctx context.Context, runData *syncer.SyncRun) error {
	cfg, err := f.decoder(runData.RunConfig)
	if err != nil {
		return fmt.Errorf("unable to decode config: %w", err)
	}
	for k, v := range f.files {
		if err := f.generate(ctx, runData, cfg, v, k); err != nil {
			return fmt.Errorf("unable to generate template for %s: %w", k, err)
		}
	}
	return nil
}

func (f *Generator) generate(ctx context.Context, runData *syncer.SyncRun, config interface{}, tmpl *template.Template, destination string) error {
	f.logger.Debug(ctx, "generating template", zap.String("destination", destination), zap.Any("config", config))
	pathDir := filepath.Dir(destination)
	if err := os.MkdirAll(pathDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", pathDir, err)
	}
	d := TemplateData{
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

func (f *Generator) Name() string {
	return f.name
}

func (f *Generator) Priority() int {
	return f.priority
}

var _ syncer.DriftSyncer = &Generator{}
