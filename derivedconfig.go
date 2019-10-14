package golog

import "sync"

type DerivedConfig struct {
	parent *Config
	filter LevelFilter
	mutex  sync.Mutex
}

func NewDerivedConfig(parent *Config, filters ...LevelFilter) *DerivedConfig {
	if parent == nil || *parent == nil {
		panic("golog.DerivedConfig parent must not be nil")
	}
	return &DerivedConfig{parent: parent, filter: LevelFilterCombine(filters...)}
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
	c.filter = LevelFilterCombine(filters...)
	c.mutex.Unlock()
}

func (c *DerivedConfig) Formatter() Formatter {
	return (*c.parent).Formatter()
}

func (c *DerivedConfig) Levels() *Levels {
	return (*c.parent).Levels()
}

func (c *DerivedConfig) IsActive(level Level) bool {
	c.mutex.Lock()
	active := c.filter.IsActive(level) && (*c.parent).IsActive(level)
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
