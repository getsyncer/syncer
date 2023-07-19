package syncer

import (
	"fmt"
	"sync"

	"go.uber.org/fx"
)

type ChildrenRegistry interface {
	Get(name string) (ChildConfig, bool)
}

type childrenRegistry struct {
	children []ChildConfig
	mu       sync.Mutex
}

func NewChildrenRegistry(children ...ChildConfig) (ChildrenRegistry, error) {
	seen := map[string]struct{}{}
	for _, s := range children {
		if _, ok := seen[s.Name]; ok {
			return nil, fmt.Errorf("child already registered: %s", s.Name)
		}
		seen[s.Name] = struct{}{}
	}
	return &childrenRegistry{
		children: children,
	}, nil
}

func (c *childrenRegistry) Get(name string) (ChildConfig, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.children == nil {
		return ChildConfig{}, false
	}
	for _, s := range c.children {
		if s.Name == name {
			return s, true
		}
	}
	return ChildConfig{}, false
}

var _ ChildrenRegistry = &childrenRegistry{}

type ChildConfig struct {
	Content []byte
	Name    string
}

func NewChildModule(name string, content []byte) fx.Option {
	constructor := func() ChildConfig {
		return ChildConfig{
			Content: content,
			Name:    name,
		}
	}
	return fx.Module(name,
		fx.Provide(
			fx.Annotate(
				constructor,
				fx.ResultTags(FxTagChildren),
			),
		),
	)
}
