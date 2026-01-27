package golog

import (
	"context"
	"sync"
)

// Ensure that DerivedConfig implements Config
var _ Config = new(DerivedConfig)

// DerivedConfig wraps another changable Config by saving a pointer
// to a variable of type Config. That variable can be changed at runtime
// so it doesn't have to be initialized at the momenht of the DerivedConfig creation.
//
// A DerivedConfig can have its own LevelFilter, which will be used to decide
// if a log message should be written or not. If the DerivedConfig has no filter,
// the filter of the parent Config will be used.
type DerivedConfig struct {
	parent           *Config
	filter           *LevelFilter
	addWriterConfigs []WriterConfig
	mutex            sync.RWMutex
}

func NewDerivedConfig(parent *Config) *DerivedConfig {
	if parent == nil || *parent == nil {
		panic("golog.DerivedConfig parent must not be nil") // Panic during setup is acceptable
	}
	return &DerivedConfig{
		parent: parent,
	}
}

func NewDerivedConfigWithFilter(parent *Config, filters ...LevelFilter) *DerivedConfig {
	if parent == nil || *parent == nil {
		panic("golog.DerivedConfig parent must not be nil") // Panic during setup is acceptable
	}
	return &DerivedConfig{
		parent: parent,
		filter: newLevelFilterOrNil(filters),
	}
}

// ConfigWithAdditionalWriterConfigs creates a new DerivedConfig with the passed writer configs
// added to the parent config. If the parent config already has the same writer configs,
// the parent config is returned unchanged.
func ConfigWithAdditionalWriterConfigs(parent *Config, configs ...WriterConfig) Config {
	if parent == nil || *parent == nil {
		panic("golog.DerivedConfig parent must not be nil") // Panic during setup is acceptable
	}
	if len(configs) == 0 {
		return *parent
	}
	parentConfigs := (*parent).WriterConfigs()
	// Combine parent and new configs, removing duplicates and nils.
	// We compare lengths to detect if new unique writers were added:
	// since we append to parentConfigs, if the deduplicated length
	// equals the parent length, all added configs were duplicates.
	combined := uniqueNonNilWriterConfigs(append(parentConfigs, configs...))
	if len(combined) == len(parentConfigs) {
		return *parent
	}
	return &DerivedConfig{
		parent:           parent,
		addWriterConfigs: combined,
	}
}

func (c *DerivedConfig) Parent() Config {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return *c.parent
}

func (c *DerivedConfig) SetParent(parent *Config) {
	if parent == nil || *parent == nil {
		panic("golog.DerivedConfig parent must not be nil") // Panic during setup is acceptable
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

func (c *DerivedConfig) SetAdditionalWriterConfigs(configs ...WriterConfig) {
	c.mutex.Lock()
	if len(configs) == 0 {
		c.addWriterConfigs = nil
	} else {
		c.addWriterConfigs = uniqueNonNilWriterConfigs(append((*c.parent).WriterConfigs(), configs...))
	}
	c.mutex.Unlock()
}

func (c *DerivedConfig) WriterConfigs() []WriterConfig {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.addWriterConfigs != nil {
		// If DerivedConfig has its own writer configs, use them
		return c.addWriterConfigs
	}
	// Else use the writer configs of the parent Config
	return (*c.parent).WriterConfigs()
}

func (c *DerivedConfig) Levels() *Levels {
	return (*c.parent).Levels()
}

func (c *DerivedConfig) IsActive(ctx context.Context, level Level) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.filter != nil {
		// If DerivedConfig has its own filter, use it
		return c.filter.IsActive(ctx, level)
	}
	// Else use the filter of the parent Config
	return (*c.parent).IsActive(ctx, level)
}

func (c *DerivedConfig) FatalLevel() Level {
	return (*c.parent).FatalLevel()
}

func (c *DerivedConfig) ErrorLevel() Level {
	return (*c.parent).ErrorLevel()
}

func (c *DerivedConfig) WarnLevel() Level {
	return (*c.parent).WarnLevel()
}

func (c *DerivedConfig) InfoLevel() Level {
	return (*c.parent).InfoLevel()
}

func (c *DerivedConfig) DebugLevel() Level {
	return (*c.parent).DebugLevel()
}

func (c *DerivedConfig) TraceLevel() Level {
	return (*c.parent).TraceLevel()
}
