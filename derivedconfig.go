package golog

import (
	"context"
)

// Ensure that DerivedConfig implements Config
var _ Config = new(DerivedConfig)

// DerivedConfig is an immutable Config that derives from a parent Config.
// The parent, filter, and writer configs are set at creation time
// and cannot be changed afterwards.
//
// Use DerivedConfig when the configuration is static after setup.
// Use [DynDerivedConfig] when the parent, filter, or writer configs
// need to be changed at runtime in a thread-safe way.
//
// A DerivedConfig can have its own LevelFilter, which will be used to decide
// if a log message should be written or not. If the DerivedConfig has no filter,
// the filter of the parent Config will be used.
type DerivedConfig struct {
	parent           *Config
	filter           *LevelFilter
	addWriterConfigs []WriterConfig
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
	return *c.parent
}

// WriterConfigs returns the writer configs for this DerivedConfig.
// If additional writer configs were set, returns those (which include the parent's).
// Otherwise, returns the parent's writer configs.
func (c *DerivedConfig) WriterConfigs() []WriterConfig {
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
