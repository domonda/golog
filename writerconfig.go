package golog

import (
	"context"
	"os"
	"slices"

	"github.com/mattn/go-isatty"
)

// WriterConfig is a factory or pool for Writers
// of a certain type and configuration.
type WriterConfig interface {
	// WriterForNewMessage returns a Writer initialized for a new message.
	// Based on the context and level the method can return
	// nil if the message should not be logged.
	WriterForNewMessage(context.Context, Level) Writer

	// FlushUnderlying flushes underlying log writing
	// streams to make sure all messages have been
	// saved or transmitted.
	// This method is intended for special circumstances like
	// before exiting the application, it's not necessary
	// to call it for every message in addtion to CommitMessage.
	FlushUnderlying()
}

var writerConfigsCtxKey int

// ContextWithAdditionalWriterConfigs returns a context
// with the passed configs uniquely added to the existing ones
// so that each WriterConfig is only added once to the context.
func ContextWithAdditionalWriterConfigs(ctx context.Context, configs ...WriterConfig) context.Context {
	// Prevent using the same writer multiple times
	configs, _ = uniqueNonNilWriterConfigs(configs)
	if len(configs) == 0 {
		return ctx
	}
	ctxConfigs := WriterConfigsFromContext(ctx)
	if len(ctxConfigs) == 0 {
		return context.WithValue(ctx, &writerConfigsCtxKey, configs)
	}
	configs, _ = uniqueNonNilWriterConfigs(append(ctxConfigs, configs...))
	return context.WithValue(ctx, &writerConfigsCtxKey, configs)
}

// WriterConfigsFromContext retrieves writer configs from the context
// that have been added with ContextWithAdditionalWriterConfigs.
func WriterConfigsFromContext(ctx context.Context) []WriterConfig {
	if configs, ok := ctx.Value(&writerConfigsCtxKey).([]WriterConfig); ok {
		return configs
	}
	return nil
}

func uniqueNonNilWriterConfigs(w []WriterConfig) (unique []WriterConfig, changed bool) {
	// Fast path checks if w can be returned as is
	numUniqueNonNil := 0
	for i, c := range w {
		if c != nil && !slices.Contains(w[:i], c) {
			numUniqueNonNil++
		}
	}
	if numUniqueNonNil == 0 {
		return nil, false
	}
	if numUniqueNonNil == len(w) {
		return w, false
	}
	// Slow path to create a new slice with unique elements
	unique = make([]WriterConfig, 0, numUniqueNonNil)
	for i, c := range w {
		if c != nil && !slices.Contains(w[:i], c) {
			unique = append(unique, c)
		}
	}
	return unique, true
}

func flushUnderlying(writer any) {
	switch x := writer.(type) {
	case interface{ Sync() error }:
		_ = x.Sync()
	}
}

// IsTerminal returns true if the current process is attached to a terminal.
func IsTerminal() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}

// DecideWriterConfigForTerminal returns terminalWriter
// if the current process is attached to a terminal,
// otherwise it returns nonTerminalWriter.
//
// This is useful for example to use a colored writer
// for terminals and a non-colored one for other outputs
// like log files.
func DecideWriterConfigForTerminal(terminalWriter WriterConfig, nonTerminalWriter WriterConfig) WriterConfig {
	if IsTerminal() {
		return terminalWriter
	}
	return nonTerminalWriter
}
