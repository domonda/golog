package golog

import (
	"context"
	"sync"
)

// DerivedConfig
type DerivedConfig struct {
	parent *Config
	filter *LevelFilter
	mutex  sync.Mutex
}

func NewDerivedConfig(parent *Config, filters ...LevelFilter) *DerivedConfig {
	if parent == nil || *parent == nil {
		panic("golog.DerivedConfig parent must not be nil")
	}
	return &DerivedConfig{
		parent: parent,
		filter: newLevelFilterOrNil(filters),
	}
}

func (c *DerivedConfig) Parent() Config {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return *c.parent
}

func (c *DerivedConfig) SetParent(parent *Config) {
	if parent == nil || *parent == nil {
		panic("golog.DerivedConfig parent must not be nil")
	}
	c.mutex.Lock()
	c.parent = parent
	c.mutex.Unlock()
}

func (c *DerivedConfig) SetFilter(filters ...LevelFilter) {
	c.mutex.Lock()
	c.filter = newLevelFilterOrNil(filters)
	c.mutex.Unlock()
}

func (c *DerivedConfig) Writer() Writer {
	return (*c.parent).Writer()
}

func (c *DerivedConfig) Levels() *Levels {
	return (*c.parent).Levels()
}

func (c *DerivedConfig) IsActive(ctx context.Context, level Level) bool {
	var active bool
	c.mutex.Lock()
	if c.filter != nil {
		// If DerivedConfig has its own filter, use it
		active = c.filter.IsActive(ctx, level)
	} else {
		// else use the filter of the parent Config
		active = (*c.parent).IsActive(ctx, level)
	}
	c.mutex.Unlock()
	return active
}

func (c *DerivedConfig) Fatal() Level {
	return (*c.parent).Fatal()
}

func (c *DerivedConfig) Error() Level {
	return (*c.parent).Error()
}

func (c *DerivedConfig) Warn() Level {
	return (*c.parent).Warn()
}

func (c *DerivedConfig) Info() Level {
	return (*c.parent).Info()
}

func (c *DerivedConfig) Debug() Level {
	return (*c.parent).Debug()
}

func (c *DerivedConfig) Trace() Level {
	return (*c.parent).Trace()
}
