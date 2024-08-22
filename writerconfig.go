package golog

import (
	"context"
	"slices"
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
	if len(configs) == 0 {
		return ctx
	}
	if c := WriterConfigsFromContext(ctx); len(c) > 0 {
		configs = append(c, configs...)
	}
	return context.WithValue(ctx, &writerConfigsCtxKey, uniqueWriterConfigs(configs))
}

// WriterConfigsFromContext retrieves writer configs from the context
// that have been added with ContextWithAdditionalWriterConfigs.
func WriterConfigsFromContext(ctx context.Context) []WriterConfig {
	if configs, ok := ctx.Value(&writerConfigsCtxKey).([]WriterConfig); ok {
		return configs
	}
	return nil
}

func uniqueWriterConfigs(w []WriterConfig) []WriterConfig {
	// Fast path checks if w can be returned as is
	numOK := 0
	for i, c := range w {
		if c != nil && !slices.Contains(w[:i], c) {
			numOK++
		}
	}
	switch numOK {
	case 0:
		return nil
	case len(w):
		return w
	}
	// Slow path to create a new slice with unique elements
	unique := make([]WriterConfig, 0, numOK)
	for i, c := range w {
		if c != nil && !slices.Contains(w[:i], c) {
			unique = append(unique, c)
		}
	}
	return unique
}

func flushUnderlying(writer any) {
	switch x := writer.(type) {
	case interface{ Sync() error }:
		_ = x.Sync()
	}
}
