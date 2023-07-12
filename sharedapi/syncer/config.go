package syncer

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ConfigLogic struct {
	Source string `yaml:"source"`
}

type ConfigSyncs struct {
	Logic  string    `yaml:"logic"`
	Config yaml.Node `yaml:"config"`
}

type RootConfig struct {
	Logic []ConfigLogic `yaml:"logic"`
	Syncs []ConfigSyncs `yaml:"syncs"`
}

type DefaultConfigLoader struct {
	filename string
}

type ConfigLoader interface {
	LoadConfig() (*RootConfig, error)
}

var _ ConfigLoader = &DefaultConfigLoader{}

func (c *DefaultConfigLoader) findConfigFile() (string, error) {
	if c.filename != "" {
		return c.filename, nil
	}
	possibleLocations := []string{
		".syncer/config.yaml",
		".syncer.yaml",
	}
	for _, loc := range possibleLocations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}
	return "", fmt.Errorf("no config file found")
}

func (c *DefaultConfigLoader) LoadConfig() (*RootConfig, error) {
	fileName, err := c.findConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to find config file: %w", err)
	}
	content, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var root RootConfig
	if err := yaml.Unmarshal(content, &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}
	return &root, nil
}
