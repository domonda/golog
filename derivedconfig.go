package golog

import "sync"

type DerivedConfig struct {
	parent    *Config
	filter    LevelFilter
	filterMtx sync.Mutex
}

func NewDerivedConfig(parent *Config, filters ...LevelFilter) *DerivedConfig {
	return &DerivedConfig{parent: parent, filter: LevelFilterCombine(filters...)}
}

func (c *DerivedConfig) SetFilter(filters ...LevelFilter) {
	c.filterMtx.Lock()
	c.filter = LevelFilterCombine(filters...)
	c.filterMtx.Unlock()
}

func (c *DerivedConfig) Formatter() Formatter {
	return (*c.parent).Formatter()
}

func (c *DerivedConfig) Levels() *Levels {
	return (*c.parent).Levels()
}

func (c *DerivedConfig) IsActive(level Level) bool {
	c.filterMtx.Lock()
	active := (*c.parent).IsActive(level) && c.filter.IsActive(level)
	c.filterMtx.Unlock()
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
