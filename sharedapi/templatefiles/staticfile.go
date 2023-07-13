package templatefiles

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/cresta/syncer/sharedapi/syncer"
	"go.uber.org/fx"
)

type TemplateData struct {
	RunData *syncer.SyncRun
	Config  interface{}
}

func NewGenerator(files map[string]*template.Template, name string, priority int, decoder Decoder) *Generator {
	return &Generator{
		files:    files,
		name:     name,
		priority: priority,
		decoder:  decoder,
	}
}

type Decoder func(syncer.RunConfig) (interface{}, error)

func NewModule(name string, files map[string]*template.Template, priority int, decoder Decoder) fx.Option {
	constructor := func() *Generator {
		return NewGenerator(files, name, priority, decoder)
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

func (f *Generator) generate(_ context.Context, runData *syncer.SyncRun, config interface{}, tmpl template.Template, destination string) error {
	pathDir := filepath.Dir(destination)
	if err := os.MkdirAll(pathDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", pathDir, err)
	}
	d := TemplateData{
		RunData: runData,
		Config:  config,
	}
	var into bytes.Buffer
	if err := tmpl.Execute(&into, d); err != nil {
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
