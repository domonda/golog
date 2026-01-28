# Golog Memory Allocation Analysis

This document analyzes where memory allocations occur in golog and compares them to other logging libraries.

## Summary

Golog currently has **2 allocations per log message** totaling **~1050 bytes/op**:

1. **~1024 bytes** - JSONWriter buffer allocation
2. **~26 bytes** - Writer slice operations in mempool

## Detailed Analysis

### Allocation #1: JSONWriter Buffer (~96% of allocations)

**Location**: `jsonwriter.go:44`

```go
func (c *JSONWriterConfig) WriterForNewMessage(ctx context.Context, level Level) Writer {
    if c.filter.IsInactive(ctx, level) {
        return nil
    }
    w := jsonWriterPool.GetOrNew()
    w.config = c
    if w.buf == nil {
        w.buf = make([]byte, 0, 1024)  // <-- ALLOCATION HAPPENS HERE
    }
    return w
}
```

**Root Cause**: The `mempool.Pointer[T].ClearAndPutBack()` method zeros the entire struct:

```go
// mempool/pointer.go:52-60
func (p *Pointer[T]) ClearAndPutBack(ptr *T) {
    if onPutBack != nil {
        onPutBack(ptr)
    }
    if ptr != nil {
        var zero T
        *ptr = zero  // <-- This zeros the entire struct, including buf field
        p.pool.Put(ptr)
    }
}
```

When `JSONWriter.buf` is set to `nil` (its zero value), the next call to `WriterForNewMessage`
allocates a new 1024-byte buffer because `w.buf == nil`.

**Why this happens**:
1. Message logging completes
2. `JSONWriter.CommitMessage()` is called
3. `jsonWriterPool.ClearAndPutBack(w)` zeros the entire `JSONWriter` struct
4. The `buf []byte` field becomes `nil`
5. Next message retrieves the pooled writer
6. Check finds `w.buf == nil`
7. New 1024-byte slice is allocated

### Allocation #2: Writer Slice Operations (~2% of allocations)

**Location**: `mempool/slice.go:80`

```go
func (p *Slice[T]) ClearAndPutBack(slice []T) {
    if onPutBack != nil {
        onPutBack(slice)
    }
    if slice != nil {
        clear(slice[:cap(slice)]) // Clear complete capacity of the slice
        p.pool.Put(slice[:0])      // <-- Small allocation during clear/put
    }
}
```

This is a smaller allocation related to managing the writers slice pool.

## Performance Impact

### Current Performance (per benchmark results)

| Scenario | ns/op | B/op | allocs/op |
|----------|-------|------|-----------|
| Simple Message | 1032 | 1050 | 2 |
| With Fields | 1377 | 1050 | 2 |
| Many Fields | 1758 | 1050 | 2 |
| Accumulated Context | 1530 | 1076 | 3 |
| **Disabled Logging** | **79** | **0** | **0** |

**Key Observation**: When logging is disabled, golog achieves **zero allocations**, which is excellent.

### Comparison with Other Libraries

| Library | Simple Message | With Fields | Disabled |
|---------|---------------|-------------|----------|
| **zerolog** | 0 B/op, 0 allocs | 0 B/op, 0 allocs | 0 B/op, 0 allocs |
| **zap** | 0 B/op, 0 allocs | 256 B/op, 1 alloc | 128 B/op, 1 alloc |
| **slog** | 0 B/op, 0 allocs | 208 B/op, 9 allocs | 48 B/op, 3 allocs |
| **golog** | 1050 B/op, 2 allocs | 1050 B/op, 2 allocs | 0 B/op, 0 allocs |
| **logrus** | 882 B/op, 20 allocs | 1933 B/op, 34 allocs | 528 B/op, 6 allocs |

## Optimization Opportunities

### Option 1: Preserve Buffer in Pool (Recommended)

**Problem**: `ClearAndPutBack` zeros the entire struct, losing the allocated buffer.

**Solution**: Don't clear the `buf` field, just reset its length:

**Current approach**:
```go
func (w *JSONWriter) CommitMessage() {
    // ... write logic ...
    w.buf = w.buf[:0]                    // Reset length but keep capacity
    jsonWriterPool.ClearAndPutBack(w)    // This zeros buf to nil!
}
```

**Potential fix approach 1** - Custom clear method:
```go
func (w *JSONWriter) clear() {
    w.config = nil
    w.buf = w.buf[:0]  // Keep the buffer, just reset length
}

func (w *JSONWriter) CommitMessage() {
    // ... write logic ...
    w.clear()
    jsonWriterPool.PutBack(w)  // New method that doesn't zero
}
```

**Potential fix approach 2** - Check buffer capacity instead of nil:
```go
func (c *JSONWriterConfig) WriterForNewMessage(ctx context.Context, level Level) Writer {
    if c.filter.IsInactive(ctx, level) {
        return nil
    }
    w := jsonWriterPool.GetOrNew()
    w.config = c
    if cap(w.buf) == 0 {  // Check capacity instead of nil
        w.buf = make([]byte, 0, 1024)
    } else {
        w.buf = w.buf[:0]  // Reuse existing buffer
    }
    return w
}
```

**Expected improvement**:
- Reduce from 2 allocs to 1 alloc per message
- Reduce from ~1050 B/op to ~26 B/op
- **~40x reduction in bytes allocated**

### Option 2: Pre-allocate Pool

Pre-warm the pool with N writers that already have buffers allocated:

```go
func init() {
    // Pre-allocate 100 writers with buffers
    for i := 0; i < 100; i++ {
        w := &JSONWriter{
            buf: make([]byte, 0, 1024),
        }
        jsonWriterPool.pool.Put(w)
    }
}
```

**Pros**: First N concurrent log calls won't allocate
**Cons**: Doesn't solve the fundamental issue; allocations still happen after clearing

### Option 3: Increase Buffer Size

Currently using 1024 bytes. If logs are typically smaller, reduce buffer size:

```go
w.buf = make([]byte, 0, 512)  // Reduce from 1024 to 512
```

**Impact**: Reduces bytes allocated but doesn't reduce allocation count.

### Option 4: Zero-Copy Buffer Management (Advanced)

Use a custom buffer pool similar to zerolog's approach:

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 0, 1024)
    },
}

func (c *JSONWriterConfig) WriterForNewMessage(ctx context.Context, level Level) Writer {
    w := jsonWriterPool.GetOrNew()
    w.buf = bufferPool.Get().([]byte)[:0]
    return w
}

func (w *JSONWriter) CommitMessage() {
    // ... write logic ...
    bufferPool.Put(w.buf)
    w.buf = nil
    jsonWriterPool.ClearAndPutBack(w)
}
```

**Expected improvement**: Near-zero allocations like zerolog

## Why Zerolog Has Zero Allocations

Zerolog's approach:
1. **Separate buffer pool**: Buffers are pooled separately from writer structs
2. **Lazy evaluation**: Only writes when event is sent
3. **Stack allocation**: Small events can stay on stack
4. **No clearing overhead**: Minimal cleanup needed

## Why Golog's Approach Is Still Valid

Despite the allocations, golog offers:

1. **Zero allocations when disabled** - Critical for production debug logging
2. **Flexible architecture** - Multiple writers, filters, formatters
3. **Consistent performance** - Predictable ~1050B regardless of field count
4. **Reasonable absolute performance** - 1-2 microseconds per log is fast enough for most use cases
5. **Better than logrus** - Fewer allocations and faster

## Recommendations

### For Library Users

**When golog is appropriate**:
- Your application logs < 100K msgs/sec
- You value features and flexibility over raw speed
- You need multiple output formats/writers
- Zero allocations when disabled is important (debug logging in production)

**When to consider alternatives**:
- Ultra high-throughput logging (> 1M msgs/sec)
- Every microsecond and byte matters
- Simple logging needs without complex routing

### For Library Maintainers

**High Priority** (Easy win):
1. Implement Option 1 (preserve buffer in pool) - Could reduce allocations by ~40x

**Medium Priority**:
2. Consider separate buffer pooling (Option 4) for zero-allocation path
3. Benchmark and tune default buffer size (1024 bytes)

**Low Priority**:
4. Pre-allocation pool (Option 2) - Marginal benefit

## Testing the Fix

After implementing optimizations, re-run benchmarks:

```bash
# Before
go test -bench=BenchmarkSimpleMessage/golog -benchmem
# BenchmarkSimpleMessage/golog-16   1201732   1032 ns/op   1050 B/op   2 allocs/op

# Expected after Option 1
# BenchmarkSimpleMessage/golog-16   ???????   ??? ns/op    ~26 B/op    1 alloc/op

# Expected after Option 4 (ideal)
# BenchmarkSimpleMessage/golog-16   ???????   ??? ns/op    0 B/op      0 allocs/op
```

Profile again to verify:
```bash
go test -bench=BenchmarkSimpleMessage/golog -memprofile=mem_after.prof
go tool pprof -top -alloc_space mem_after.prof
```

## Conclusion

Golog's 2 allocations per message (1050 bytes) are primarily due to:
1. **Buffer re-allocation** (96%) - Easily fixable by preserving buffer capacity in pool
2. **Slice operations** (2%) - Minor overhead from pool management

The disabled logging path already achieves zero allocations, demonstrating the code
can be allocation-free when optimized. With buffer pooling improvements, golog could
achieve near-zero allocations while maintaining its flexible architecture.

Current performance is reasonable for most applications, but there's clear room for
40x+ improvement in allocation efficiency with straightforward optimizations.
