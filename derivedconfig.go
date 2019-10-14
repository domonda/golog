package golog

import "sync"

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
	return &DerivedConfig{parent: parent, filter: newLevelFilterOrNil(filters)}
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

func (c *DerivedConfig) Formatter() Formatter {
	return (*c.parent).Formatter()
}

func (c *DerivedConfig) Levels() *Levels {
	return (*c.parent).Levels()
}

func (c *DerivedConfig) IsActive(level Level) bool {
	var active bool
	c.mutex.Lock()
	if c.filter != nil {
		active = (*c.filter).IsActive(level)
	} else {
		active = (*c.parent).IsActive(level)
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
