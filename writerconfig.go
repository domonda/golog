package golog

import (
	"context"
	"os"
	"slices"

	"golang.org/x/term"
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
	configs = uniqueNonNilWriterConfigs(configs)
	if len(configs) == 0 {
		return ctx
	}

	if ctxConfigs := WriterConfigsFromContext(ctx); len(ctxConfigs) > 0 {
		configs = mergeWriterConfigs(ctxConfigs, configs)
	}
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

// uniqueNonNilWriterConfigs returns a slice containing only the unique,
// non-nil elements from w, preserving their original order.
// Returns nil if w contains no non-nil elements.
// Returns w unchanged if all elements are already unique and non-nil.
func uniqueNonNilWriterConfigs(w []WriterConfig) []WriterConfig {
	switch len(w) {
	case 0:
		return nil
	case 1:
		if w[0] == nil {
			return nil
		}
		return w
	}
	// Find first nil or duplicate element.
	// Everything before it is guaranteed clean.
	firstBad := -1
	for i, c := range w {
		if c == nil || slices.Contains(w[:i], c) {
			firstBad = i
			break
		}
	}
	if firstBad < 0 {
		return w // Already clean
	}
	// Single pass: copy clean prefix, then filter the rest
	unique := make([]WriterConfig, firstBad, len(w))
	copy(unique, w[:firstBad])
	for _, c := range w[firstBad:] {
		if c != nil && !slices.Contains(unique, c) {
			unique = append(unique, c)
		}
	}
	if len(unique) == 0 {
		return nil
	}
	return unique
}

// mergeWriterConfigs returns the unique, non-nil writer configs from both slices.
// Returns a unchanged if it contains no nils or internal duplicates
// and already contains all non-nil elements from b.
func mergeWriterConfigs(a, b []WriterConfig) []WriterConfig {
	if len(b) == 0 {
		return uniqueNonNilWriterConfigs(a)
	}
	if len(a) == 0 {
		return uniqueNonNilWriterConfigs(b)
	}
	// Check if b adds any new non-nil elements not already in a
	hasNew := false
	for _, c := range b {
		if c != nil && !slices.Contains(a, c) {
			hasNew = true
			break
		}
	}
	if !hasNew {
		// b adds nothing new, just ensure a itself is clean
		return uniqueNonNilWriterConfigs(a)
	}
	// Build merged result without intermediate append(a, b...) allocation
	result := make([]WriterConfig, 0, len(a)+len(b))
	for i, c := range a {
		if c != nil && !slices.Contains(a[:i], c) {
			result = append(result, c)
		}
	}
	for _, c := range b {
		if c != nil && !slices.Contains(result, c) {
			result = append(result, c)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// flushUnderlying flushes any buffered data in writer by calling
// Sync() if the writer implements it. Sync errors are ignored.
func flushUnderlying(writer any) {
	switch x := writer.(type) {
	case interface{ Sync() error }:
		_ = x.Sync()
	}
}

// IsTerminal returns true if the current process is attached to a terminal.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
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
