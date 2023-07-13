package syncer

import (
	"context"
	"fmt"
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
	Name   string    `yaml:"name"`
	Config yaml.Node `yaml:"config"`
}

type RootConfig struct {
	Version int           `yaml:"version"`
	Config  yaml.Node     `yaml:"config"`
	Logic   []ConfigLogic `yaml:"logic"`
	Syncs   []ConfigSyncs `yaml:"syncs"`
}

type DefaultConfigLoader struct {
	filename string
}

func NewDefaultConfigLoader() *DefaultConfigLoader {
	return &DefaultConfigLoader{}
}

type ConfigLoader interface {
	LoadConfig(ctx context.Context, filename string) (*RootConfig, error)
}

var _ ConfigLoader = &DefaultConfigLoader{}

func (c *DefaultConfigLoader) findConfigFile() (string, error) {
	if c.filename != "" {
		return c.filename, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return DefaultFindConfigFile(wd)
}

func DefaultFindConfigFile(wd string) (string, error) {
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

func (c *DefaultConfigLoader) LoadConfig(_ context.Context, filename string) (*RootConfig, error) {
	if filename == "" {
		fileName, err := c.findConfigFile()
		if err != nil {
			return nil, fmt.Errorf("failed to find config file: %w", err)
		}
		filename = fileName
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var root RootConfig
	if err := yaml.Unmarshal(content, &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}
	if root.Version != 1 {
		return nil, fmt.Errorf("unknown config version: %d", root.Version)
	}
	return &root, nil
}
