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

// NewDerivedConfig creates a new DerivedConfig that wraps the parent config.
// Panics if parent is nil.
func NewDerivedConfig(parent *Config) *DerivedConfig {
	if parent == nil || *parent == nil {
		panic("golog.DerivedConfig parent must not be nil") // Panic during setup is acceptable
	}
	return &DerivedConfig{
		parent: parent,
	}
}

// NewDerivedConfigWithFilter creates a new DerivedConfig with its own level filter.
// The filters are combined using JoinLevelFilters.
// Panics if parent is nil.
func NewDerivedConfigWithFilter(parent *Config, filters ...LevelFilter) *DerivedConfig {
	if parent == nil || *parent == nil {
		panic("golog.DerivedConfig parent must not be nil") // Panic during setup is acceptable
	}
	return &DerivedConfig{
		parent: parent,
		filter: newLevelFilterOrNil(filters),
	}
}

// ConfigWithAdditionalWriterConfigs returns a Config with the passed writer configs
// added to the parent config.
//
// Returns the parent config unchanged if:
//   - no configs are passed, or
//   - all passed configs are duplicates of configs already in the parent
//
// Returns a new DerivedConfig if any new unique writer configs are added.
// Duplicate and nil configs are automatically removed.
// Panics if parent is nil.
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

// Parent returns the current parent Config.
// The returned Config is the one that was set at creation or via SetParent.
func (c *DerivedConfig) Parent() Config {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return *c.parent
}

// SetParent changes the parent Config at runtime.
// Panics if parent is nil or points to a nil Config.
func (c *DerivedConfig) SetParent(parent *Config) {
	if parent == nil || *parent == nil {
		panic("golog.DerivedConfig parent must not be nil") // Panic during setup is acceptable
	}
	c.mutex.Lock()
	c.parent = parent
	c.mutex.Unlock()
}

// SetFilter sets the DerivedConfig's own level filter.
// Multiple filters are combined using JoinLevelFilters.
// Pass no arguments to clear the filter and use the parent's filter.
func (c *DerivedConfig) SetFilter(filters ...LevelFilter) {
	c.mutex.Lock()
	c.filter = newLevelFilterOrNil(filters)
	c.mutex.Unlock()
}

// SetAdditionalWriterConfigs sets writer configs to be used in addition to the parent's.
// The configs are combined with the parent's writer configs, with duplicates removed.
// Pass no arguments to clear additional writers and use only the parent's writers.
func (c *DerivedConfig) SetAdditionalWriterConfigs(configs ...WriterConfig) {
	c.mutex.Lock()
	if len(configs) == 0 {
		c.addWriterConfigs = nil
	} else {
		c.addWriterConfigs = uniqueNonNilWriterConfigs(append((*c.parent).WriterConfigs(), configs...))
	}
	c.mutex.Unlock()
}

// WriterConfigs returns the writer configs for this DerivedConfig.
// If additional writer configs were set, returns those (which include the parent's).
// Otherwise, returns the parent's writer configs.
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

// Levels returns the Levels from the parent Config.
// DerivedConfig does not have its own Levels.
func (c *DerivedConfig) Levels() *Levels {
	return (*c.parent).Levels()
}

// IsActive returns whether logging is active for the given level and context.
// If DerivedConfig has its own filter, that filter is used.
// Otherwise, the parent's IsActive method is called.
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

// FatalLevel returns the fatal level from the parent Config.
func (c *DerivedConfig) FatalLevel() Level {
	return (*c.parent).FatalLevel()
}

// ErrorLevel returns the error level from the parent Config.
func (c *DerivedConfig) ErrorLevel() Level {
	return (*c.parent).ErrorLevel()
}

// WarnLevel returns the warn level from the parent Config.
func (c *DerivedConfig) WarnLevel() Level {
	return (*c.parent).WarnLevel()
}

// InfoLevel returns the info level from the parent Config.
func (c *DerivedConfig) InfoLevel() Level {
	return (*c.parent).InfoLevel()
}

// DebugLevel returns the debug level from the parent Config.
func (c *DerivedConfig) DebugLevel() Level {
	return (*c.parent).DebugLevel()
}

// TraceLevel returns the trace level from the parent Config.
func (c *DerivedConfig) TraceLevel() Level {
	return (*c.parent).TraceLevel()
}
