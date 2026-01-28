# Golog Memory Allocation Analysis

This document analyzes memory allocations in golog and compares them to other logging libraries.

## Summary

Golog has been optimized to achieve **1 allocation per log message** totaling **~24 bytes/op** for most scenarios. This is a significant improvement from the previous 2 allocations totaling ~1050 bytes/op.

**Key achievement**: When logging is disabled, golog achieves **zero allocations**, which is critical for production debug logging.

## Current Performance

### Benchmark Results (Apple M2)

| Scenario | ns/op | B/op | allocs/op |
|----------|-------|------|-----------|
| Simple Message | 638 | 24 | 1 |
| With Fields (4 fields) | 1066 | 24 | 1 |
| Many Fields (10 fields) | 1741 | 24 | 1 |
| Accumulated Context | 1319 | 48 | 2 |
| Complex Fields (error, time) | 1848 | 152 | 4 |
| Text Output | 1550 | 347 | 8 |
| **Disabled Logging** | **70** | **0** | **0** |

### Comparison with Other Libraries

| Library | Simple Message | With Fields | Disabled |
|---------|---------------|-------------|----------|
| **zerolog** | 0 B/op, 0 allocs | 0 B/op, 0 allocs | 0 B/op, 0 allocs |
| **zap** | 0 B/op, 0 allocs | 256 B/op, 1 alloc | 128 B/op, 1 alloc |
| **slog** | 0 B/op, 0 allocs | 208 B/op, 9 allocs | 48 B/op, 3 allocs |
| **golog** | 24 B/op, 1 alloc | 24 B/op, 1 alloc | 0 B/op, 0 allocs |
| **logrus** | 889 B/op, 20 allocs | 1939 B/op, 34 allocs | 528 B/op, 6 allocs |

## Optimization History

### Previous Issue: JSONWriter Buffer Re-allocation

**Problem**: The `ClearAndPutBack()` method zeroed the entire `JSONWriter` struct, including the `buf` field. This caused a new 1024-byte buffer allocation on every log message.

**Solution**: Changed `CommitMessage()` to:
1. Manually clear only necessary fields
2. Preserve buffer with `w.buf = w.buf[:0]` (keeps capacity, resets length)
3. Use `PutBack()` instead of `ClearAndPutBack()` (now renamed to `ZeroAndPutBack()`)

```go
func (w *JSONWriter) CommitMessage() {
    // ... write logic ...

    // Reset and return to pool
    w.config = nil
    w.buf = w.buf[:0]          // Preserve buffer capacity
    jsonWriterPool.PutBack(w)  // Don't zero the struct
}
```

**Result**: Reduced from ~1050 B/op to ~24 B/op (~97.7% reduction in bytes allocated).

## Remaining Allocation Sources

### Allocation #1: Writers Slice (~24 bytes)

The remaining allocation comes from the writers slice pool operations. This is a minimal overhead for managing the slice of active writers.

**Location**: `mempools.go` - `writersPool`

This allocation is necessary for the flexible multi-writer architecture and represents an acceptable trade-off for the functionality it provides.

## Why Golog's Approach Is Valid

Despite having 1 allocation per message (vs zerolog's 0), golog offers:

1. **Zero allocations when disabled** - Critical for production debug logging
2. **Flexible architecture** - Multiple writers, filters, formatters
3. **Consistent performance** - Predictable ~24B regardless of field count
4. **Competitive with slog** - Standard library's structured logger
5. **Much better than logrus** - ~37x fewer bytes, ~20x fewer allocations

## When to Choose Golog

**Golog is appropriate when**:
- Your application logs < 100K msgs/sec
- You value features and flexibility
- You need multiple output formats/writers
- Zero allocations when disabled is important (debug logging in production)
- You're already in the domonda ecosystem

**Consider alternatives when**:
- Ultra high-throughput logging (> 1M msgs/sec)
- Every allocation matters (embedded systems)
- Simple logging needs without complex routing

## Running Benchmarks

```bash
# From golog directory
go test ./benchmarks -bench=. -benchmem -benchtime=1s

# Specific benchmark
go test ./benchmarks -bench=BenchmarkSimpleMessage -benchmem

# With memory profiling
go test ./benchmarks -bench=BenchmarkSimpleMessage/golog -memprofile=mem.prof
go tool pprof -top -alloc_space mem.prof
```

## References

- [README.md](README.md) - Full comparative benchmark results
- [Better Stack Go Logging Benchmarks](https://betterstack.com/community/guides/logging/best-golang-logging-libraries/)
