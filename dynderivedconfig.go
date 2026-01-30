package golog

import (
	"context"
	"sync"
)

// Ensure that DynDerivedConfig implements Config
var _ Config = new(DynDerivedConfig)

// DynDerivedConfig is a thread-safe, mutable Config that derives from a parent Config.
// The parent, filter, and writer configs can be changed at runtime
// using SetParent, SetFilter, and SetAdditionalWriterConfigs.
// All reads and writes are protected by a sync.RWMutex.
//
// Use DynDerivedConfig when the parent, filter, or writer configs
// need to be changed at runtime in a thread-safe way.
// Use [DerivedConfig] when the configuration is static after setup.
//
// A DynDerivedConfig can have its own LevelFilter, which will be used to decide
// if a log message should be written or not. If the DynDerivedConfig has no filter,
// the filter of the parent Config will be used.
type DynDerivedConfig struct {
	parent           *Config
	filter           *LevelFilter
	addWriterConfigs []WriterConfig
	mutex            sync.RWMutex
}

// NewDynDerivedConfig creates a new DynDerivedConfig that wraps the parent config.
// Panics if parent is nil.
func NewDynDerivedConfig(parent *Config) *DynDerivedConfig {
	if parent == nil || *parent == nil {
		panic("golog.DynDerivedConfig parent must not be nil") // Panic during setup is acceptable
	}
	return &DynDerivedConfig{
		parent: parent,
	}
}

// NewDynDerivedConfigWithFilter creates a new DynDerivedConfig with its own level filter.
// The filters are combined using JoinLevelFilters.
// Panics if parent is nil.
func NewDynDerivedConfigWithFilter(parent *Config, filters ...LevelFilter) *DynDerivedConfig {
	if parent == nil || *parent == nil {
		panic("golog.DynDerivedConfig parent must not be nil") // Panic during setup is acceptable
	}
	return &DynDerivedConfig{
		parent: parent,
		filter: newLevelFilterOrNil(filters),
	}
}

// Parent returns the current parent Config.
// The returned Config is the one that was set at creation or via SetParent.
func (c *DynDerivedConfig) Parent() Config {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return *c.parent
}

// SetParent changes the parent Config at runtime.
// Panics if parent is nil or points to a nil Config.
func (c *DynDerivedConfig) SetParent(parent *Config) {
	if parent == nil || *parent == nil {
		panic("golog.DynDerivedConfig parent must not be nil") // Panic during setup is acceptable
	}
	c.mutex.Lock()
	c.parent = parent
	c.mutex.Unlock()
}

// SetFilter sets the DynDerivedConfig's own level filter.
// Multiple filters are combined using JoinLevelFilters.
// Pass no arguments to clear the filter and use the parent's filter.
func (c *DynDerivedConfig) SetFilter(filters ...LevelFilter) {
	c.mutex.Lock()
	c.filter = newLevelFilterOrNil(filters)
	c.mutex.Unlock()
}

// SetAdditionalWriterConfigs sets writer configs to be used in addition to the parent's.
// The configs are combined with the parent's writer configs, with duplicates removed.
// Pass no arguments to clear additional writers and use only the parent's writers.
func (c *DynDerivedConfig) SetAdditionalWriterConfigs(configs ...WriterConfig) {
	c.mutex.Lock()
	if len(configs) == 0 {
		c.addWriterConfigs = nil
	} else {
		c.addWriterConfigs = uniqueNonNilWriterConfigs(append((*c.parent).WriterConfigs(), configs...))
	}
	c.mutex.Unlock()
}

// WriterConfigs returns the writer configs for this DynDerivedConfig.
// If additional writer configs were set, returns those (which include the parent's).
// Otherwise, returns the parent's writer configs.
func (c *DynDerivedConfig) WriterConfigs() []WriterConfig {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.addWriterConfigs != nil {
		// If DynDerivedConfig has its own writer configs, use them
		return c.addWriterConfigs
	}
	// Else use the writer configs of the parent Config
	return (*c.parent).WriterConfigs()
}

// Levels returns the Levels from the parent Config.
// DynDerivedConfig does not have its own Levels.
func (c *DynDerivedConfig) Levels() *Levels {
	return c.Parent().Levels()
}

// IsActive returns whether logging is active for the given level and context.
// If DynDerivedConfig has its own filter, that filter is used.
// Otherwise, the parent's IsActive method is called.
func (c *DynDerivedConfig) IsActive(ctx context.Context, level Level) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.filter != nil {
		// If DynDerivedConfig has its own filter, use it
		return c.filter.IsActive(ctx, level)
	}
	// Else use the filter of the parent Config
	return (*c.parent).IsActive(ctx, level)
}

// FatalLevel returns the fatal level from the parent Config.
func (c *DynDerivedConfig) FatalLevel() Level {
	return c.Parent().FatalLevel()
}

// ErrorLevel returns the error level from the parent Config.
func (c *DynDerivedConfig) ErrorLevel() Level {
	return c.Parent().ErrorLevel()
}

// WarnLevel returns the warn level from the parent Config.
func (c *DynDerivedConfig) WarnLevel() Level {
	return c.Parent().WarnLevel()
}

// InfoLevel returns the info level from the parent Config.
func (c *DynDerivedConfig) InfoLevel() Level {
	return c.Parent().InfoLevel()
}

// DebugLevel returns the debug level from the parent Config.
func (c *DynDerivedConfig) DebugLevel() Level {
	return c.Parent().DebugLevel()
}

// TraceLevel returns the trace level from the parent Config.
func (c *DynDerivedConfig) TraceLevel() Level {
	return c.Parent().TraceLevel()
}
