/*
Package goslog provides a bridge between Go's standard log/slog package and golog.

This package implements a slog.Handler that routes log records from slog to golog,
enabling applications to use the standard library's slog API while benefiting from
golog's flexible output formatting, multiple writers, and performance optimizations.

# Basic Usage

	import (
		"log/slog"

		"github.com/domonda/golog"
		"github.com/domonda/golog/goslog"
	)

	// Create a golog logger
	gologLogger := golog.NewLogger(
		golog.NewConfig(
			&golog.DefaultLevels,
			golog.AllLevelsActive,
			golog.NewJSONWriterConfig(os.Stdout, nil),
		),
	)

	// Create a slog handler that uses the golog logger
	handler := goslog.Handler(gologLogger, goslog.ConvertDefaultLevels)

	// Use it with slog
	logger := slog.New(handler)
	logger.Info("Hello from slog", "key", "value")

# Level Conversion

The package provides ConvertDefaultLevels to map slog levels to golog levels.
You can provide a custom ConvertLevelFunc if you need different mapping logic.

# Compatibility

This handler passes the slog.Handler test suite (slogtest.TestHandler) and
supports all standard slog features including:
  - Structured attributes with typed values
  - Attribute groups
  - WithAttrs and WithGroup for creating child loggers
  - LogValuer interface for lazy evaluation
*/
package goslog

import (
	"context"
	"log/slog"

	"github.com/domonda/golog"
)

// ConvertLevelFunc converts a slog.Level to a golog.Level.
// Custom conversion functions can be provided to Handler if different
// level mapping is required.
type ConvertLevelFunc func(slog.Level) golog.Level

// ConvertDefaultLevels converts slog log levels to golog levels using a standard mapping.
//
// The mapping preserves relative level differences:
//   - slog.LevelDebug and below → golog.DefaultLevels.Debug and below
//   - slog.LevelInfo → golog.DefaultLevels.Info
//   - slog.LevelWarn → golog.DefaultLevels.Warn
//   - slog.LevelError and above → golog.DefaultLevels.Error and above
//
// Levels outside the valid golog range return golog.LevelInvalid.
func ConvertDefaultLevels(l slog.Level) golog.Level {
	var i int
	switch {
	case l <= slog.LevelDebug:
		i = int(golog.DefaultLevels.Debug) - int(slog.LevelDebug-l)
	case l <= slog.LevelInfo:
		i = int(golog.DefaultLevels.Info) - int(slog.LevelInfo-l)
	case l <= slog.LevelWarn:
		i = int(golog.DefaultLevels.Warn) - int(slog.LevelWarn-l)
	case l <= slog.LevelError:
		i = int(golog.DefaultLevels.Error) - int(slog.LevelError-l)
	default:
		i = int(golog.DefaultLevels.Error) + int(l-slog.LevelError)
	}
	if i < int(golog.LevelMin) || i > int(golog.LevelMax) {
		return golog.LevelInvalid
	}
	return golog.Level(i)
}

// Handler creates a new slog.Handler that routes log records to the provided golog.Logger.
//
// The convertLevel function is used to map slog.Level values to golog.Level values.
// Use ConvertDefaultLevels for standard mapping, or provide a custom function for
// different level mapping behavior.
//
// Example:
//
//	gologLogger := golog.NewLogger(config)
//	slogHandler := goslog.Handler(gologLogger, goslog.ConvertDefaultLevels)
//	logger := slog.New(slogHandler)
//
// The handler supports all slog.Handler interface methods:
//   - Enabled: checks if the golog logger is active for the given level
//   - Handle: writes the log record to the golog logger
//   - WithAttrs: creates a child handler with additional attributes
//   - WithGroup: creates a child handler with a group prefix
func Handler(logger *golog.Logger, convertLevel ConvertLevelFunc) slog.Handler {
	return &handler{logger: logger, convertLevel: convertLevel}
}

// handler implements slog.Handler by routing to a golog.Logger.
type handler struct {
	logger       *golog.Logger      // The underlying golog logger
	convertLevel ConvertLevelFunc   // Function to convert slog levels to golog levels
	attrs        []slog.Attr        // Pre-configured attributes (from WithAttrs)
	groupPrefix  string             // Current group prefix (from WithGroup)
}

// clone creates a shallow copy of the handler.
// Used by WithAttrs and WithGroup to create child handlers.
func (h *handler) clone() *handler {
	return &handler{
		logger:       h.logger,
		convertLevel: h.convertLevel,
		attrs:        h.attrs,
		groupPrefix:  h.groupPrefix,
	}
}

// Enabled reports whether the handler handles records at the given level.
// This method is part of the slog.Handler interface.
//
// It delegates to the golog logger's IsActive method after converting
// the slog level to a golog level.
func (h *handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.logger.IsActive(ctx, h.convertLevel(level))
}

// Handle processes a slog.Record by writing it to the golog logger.
// This method is part of the slog.Handler interface.
//
// The record's attributes are written to the golog message, with any
// pre-configured attributes (from WithAttrs) written first, followed
// by the record's own attributes. Group prefixes from WithGroup are
// applied to attribute keys.
func (h *handler) Handle(ctx context.Context, record slog.Record) error {
	msg := h.logger.NewMessageAt(ctx, record.Time, h.convertLevel(record.Level), record.Message)
	for _, a := range h.attrs {
		msg = writeAttr(msg, "", a.Key, a.Value)
	}
	record.Attrs(func(a slog.Attr) bool {
		msg = writeAttr(msg, h.groupPrefix, a.Key, a.Value)
		return true
	})
	msg.Log()
	return nil
}

// WithAttrs returns a new handler with additional attributes.
// This method is part of the slog.Handler interface.
//
// The returned handler will include the given attributes in every log record.
// If the handler has a group prefix (from WithGroup), the attributes will be
// prefixed with that group name.
//
// Example:
//
//	handler2 := handler1.WithAttrs([]slog.Attr{
//	    slog.String("service", "api"),
//	    slog.Int("version", 2),
//	})
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	with := h.clone()
	if h.groupPrefix != "" {
		for i := range attrs {
			attrs[i].Key = prefixKey(h.groupPrefix, attrs[i].Key)
		}
	}
	with.attrs = append(with.attrs, attrs...)
	return with
}

// WithGroup returns a new handler with a group prefix.
// This method is part of the slog.Handler interface.
//
// All attributes logged by the returned handler will have their keys
// prefixed with the group name. Multiple groups can be nested by calling
// WithGroup multiple times.
//
// Example:
//
//	handler2 := handler1.WithGroup("request")
//	// Attributes will be prefixed with "request.", e.g., "request.method"
func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	with := h.clone()
	with.groupPrefix = prefixKey(with.groupPrefix, name)
	return with
}

// writeAttr writes a slog attribute to a golog message.
//
// It handles all slog value kinds including:
//   - Primitive types (bool, int64, uint64, float64, string, time, duration)
//   - Groups (recursively processes nested attributes)
//   - LogValuer (resolves lazy values)
//   - Any (for untyped values)
//
// Group names are prefixed to attribute keys using dot notation (e.g., "group.key").
func writeAttr(m *golog.Message, group, key string, value slog.Value) *golog.Message {
	kind := value.Kind()
	if kind == slog.KindGroup {
		group = prefixKey(group, key)
		for _, attr := range value.Group() {
			m = writeAttr(m, group, attr.Key, attr.Value)
		}
		return m
	}
	if key == "" {
		return m
	}
	switch kind {
	case slog.KindAny:
		return m.Any(prefixKey(group, key), value.Any())
	case slog.KindBool:
		return m.Bool(prefixKey(group, key), value.Bool())
	case slog.KindDuration:
		return m.Duration(prefixKey(group, key), value.Duration())
	case slog.KindFloat64:
		return m.Float(prefixKey(group, key), value.Float64())
	case slog.KindInt64:
		return m.Int64(prefixKey(group, key), value.Int64())
	case slog.KindString:
		return m.Str(prefixKey(group, key), value.String())
	case slog.KindTime:
		return m.Time(prefixKey(group, key), value.Time())
	case slog.KindUint64:
		return m.Uint64(prefixKey(group, key), value.Uint64())
	case slog.KindLogValuer:
		return writeAttr(m, group, key, value.LogValuer().LogValue().Resolve())
	default:
		// Should never happen, but don't panic and still log any value
		return m.Any(prefixKey(group, key), value)
	}
}

// prefixKey combines a group name and a key with dot notation.
//
// If either group or key is empty, returns the non-empty value.
// If both are non-empty, returns "group.key".
// This is used to create hierarchical attribute names for slog groups.
func prefixKey(group, key string) string {
	if group == "" {
		return key
	}
	if key == "" {
		return group
	}
	return group + "." + key
}
