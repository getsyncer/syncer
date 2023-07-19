package syncer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ConfigLogic struct {
	Source string `yaml:"source"`
}

func (c *ConfigLogic) SourceWithoutVersion() string {
	parts := strings.SplitN(c.Source, "@", 2)
	if len(parts) == 1 {
		return c.Source
	}
	return parts[0]
}

func (c *ConfigLogic) SourceVersion() string {
	parts := strings.SplitN(c.Source, "@", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

type ConfigSyncs struct {
	Logic  string    `yaml:"logic"`
	Id     string    `yaml:"id"`
	Config RunConfig `yaml:"config"`
}

type RootConfig struct {
	Children []ConfigLogic `yaml:"children"`
	Version  int           `yaml:"version"`
	Config   RunConfig     `yaml:"config"`
	Logic    []ConfigLogic `yaml:"logic"`
	Syncs    []ConfigSyncs `yaml:"syncs"`
}

func (r *RootConfig) AsYaml() (string, error) {
	var into bytes.Buffer
	enc := yaml.NewEncoder(&into)
	enc.SetIndent(2)
	if err := enc.Encode(r); err != nil {
		return "", fmt.Errorf("error encoding yaml: %w", err)
	}
	return into.String(), nil
}

type DefaultConfigLoader struct {
	childrenRegistry ChildrenRegistry
}

func NewDefaultConfigLoader(childrenRegistry ChildrenRegistry) *DefaultConfigLoader {
	return &DefaultConfigLoader{
		childrenRegistry: childrenRegistry,
	}
}

func (r *RootConfig) Merge(other *RootConfig) error {
	if other == nil {
		return nil
	}
	if other.Version != r.Version {
		return fmt.Errorf("cannot merge different config versions: %d vs %d", r.Version, other.Version)
	}
	if other.Children != nil {
		r.Children = append(r.Children, other.Children...)
	}
	if other.Logic != nil {
		r.Logic = append(r.Logic, other.Logic...)
	}
	if other.Syncs != nil {
		r.Syncs = append(r.Syncs, other.Syncs...)
	}
	if err := r.Config.Merge(other.Config); err != nil {
		return fmt.Errorf("failed to merge config: %w", err)
	}
	return nil
}

type ConfigLoader interface {
	LoadConfig(ctx context.Context, contents io.Reader) (*RootConfig, error)
	FlattenChildren(ctx context.Context, root *RootConfig) error
}

var _ ConfigLoader = &DefaultConfigLoader{}

func DefaultFindConfigFile(wd string) (string, error) {
	if wd == "" {
		var err error
		wd, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
	}
	possibleLocations := []string{
		".syncer/config.yaml",
		".syncer.yaml",
	}
	for _, loc := range possibleLocations {
		fileLoc := filepath.Join(wd, loc)
		if _, err := os.Stat(fileLoc); err == nil {
			return loc, nil
		}
	}
	return "", fmt.Errorf("no config file found")
}

func ConfigFromFile(ctx context.Context, loader ConfigLoader) (*RootConfig, error) {
	f, err := DefaultFindConfigFile("")
	if err != nil {
		return nil, fmt.Errorf("failed to find config file: %w", err)
	}
	b, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	return loader.LoadConfig(ctx, bytes.NewReader(b))
}

func (c *DefaultConfigLoader) LoadConfig(_ context.Context, contents io.Reader) (*RootConfig, error) {
	var root RootConfig
	dec := yaml.NewDecoder(contents)
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}
	if root.Version != 1 {
		return nil, fmt.Errorf("unknown config version: %d", root.Version)
	}
	return &root, nil
}

func (c *DefaultConfigLoader) FlattenChildren(_ context.Context, root *RootConfig) error {
	// Now process children
	processedChildrenSources := make(map[string]struct{})
	i := 0
	for len(root.Children) > 0 {
		i++
		if i > 1000 {
			return fmt.Errorf("too many children: forever loop protection")
		}
		originalChildren := root.Children
		root.Children = nil
		for _, child := range originalChildren {
			content, exists := c.childrenRegistry.Get(child.Source)
			if !exists {
				return fmt.Errorf("unknown child: %s", child.Source)
			}
			if _, exists := processedChildrenSources[child.Source]; exists {
				continue
			}
			processedChildrenSources[child.Source] = struct{}{}
			// Now unmarshal the child
			var childRoot RootConfig
			if err := yaml.Unmarshal(content.Content, &childRoot); err != nil {
				return fmt.Errorf("failed to unmarshal child: %w", err)
			}
			if err := root.Merge(&childRoot); err != nil {
				return fmt.Errorf("failed to merge child: %w", err)
			}
		}
	}
	return nil
}
