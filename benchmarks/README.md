# Golog Comparative Benchmarks

This document contains comparative benchmarks between golog and other popular Go structured logging libraries.

## Libraries Compared

- **golog** - This library
- **zerolog** - High-performance structured logging
- **zap** - Uber's fast, structured, leveled logger
- **slog** - Go standard library's structured logging (Go 1.21+)
- **logrus** - Popular structured logger

## Running the Benchmarks

The benchmarks are in a separate module at `benchmarks/` to avoid adding dependencies on other
logging libraries (zerolog, zap, logrus) to the main golog module. The benchmarks module uses
a `replace` directive to reference the local golog package.

To run all comparative benchmarks from the golog directory:

```bash
go test ./benchmarks -bench=. -benchmem -benchtime=1s
```

Or from the benchmarks directory:

```bash
cd benchmarks
go test -bench=. -benchmem -benchtime=1s
```

To run a specific benchmark:

```bash
go test ./benchmarks -bench=BenchmarkSimpleMessage -benchmem
```

## Benchmark Scenarios

The benchmarks test several common logging scenarios:

### 1. BenchmarkSimpleMessage
Tests logging a simple message without any structured fields.
- **Use case**: Basic event logging
- **What it measures**: Baseline performance overhead

### 2. BenchmarkWithFields
Tests logging a message with 4 structured fields (string, int, float, bool).
- **Use case**: Standard structured logging
- **What it measures**: Field serialization performance

### 3. BenchmarkWithManyFields
Tests logging a message with 10 structured fields.
- **Use case**: Rich contextual logging
- **What it measures**: Performance with complex log entries

### 4. BenchmarkWithAccumulatedContext
Tests logging with pre-configured logger context.
- **Use case**: Request-scoped logging with common fields
- **What it measures**: Sub-logger and context handling efficiency

### 5. BenchmarkDisabled
Tests the overhead when logging at a disabled level.
- **Use case**: Debug logging in production
- **What it measures**: Branch prediction and no-op performance

### 6. BenchmarkComplexFields
Tests logging with complex types (errors, timestamps).
- **Use case**: Error logging and time-based events
- **What it measures**: Complex type serialization

### 7. BenchmarkTextOutput
Tests text/console output format instead of JSON.
- **Use case**: Human-readable development logs
- **What it measures**: Text formatting performance

## Sample Results

Results from a sample run on Intel Xeon Platinum 8581C @ 2.10GHz:

```
BenchmarkSimpleMessage/golog-16       	 1201732	      1032 ns/op	    1050 B/op	       2 allocs/op
BenchmarkSimpleMessage/zerolog-16     	16943080	        74.92 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/zap-16         	 3537788	       343.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/slog-16        	 2385986	       491.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/logrus-16      	  728607	      1623 ns/op	     882 B/op	      20 allocs/op

BenchmarkWithFields/golog-16          	  799130	      1377 ns/op	    1050 B/op	       2 allocs/op
BenchmarkWithFields/zerolog-16        	 5594331	       211.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithFields/zap-16            	 1765508	       691.5 ns/op	     256 B/op	       1 allocs/op
BenchmarkWithFields/slog-16           	  924662	      1289 ns/op	     208 B/op	       9 allocs/op
BenchmarkWithFields/logrus-16         	  337856	      3595 ns/op	    1933 B/op	      34 allocs/op

BenchmarkDisabled/golog-16            	14264186	        79.43 ns/op	       0 B/op	       0 allocs/op
BenchmarkDisabled/zerolog-16          	272754601	         4.346 ns/op	       0 B/op	       0 allocs/op
BenchmarkDisabled/zap-16              	12929640	        89.96 ns/op	     128 B/op	       1 allocs/op
BenchmarkDisabled/slog-16             	13903776	        81.30 ns/op	      48 B/op	       3 allocs/op
BenchmarkDisabled/logrus-16           	 2224910	       537.5 ns/op	     528 B/op	       6 allocs/op
```

## Performance Analysis

### Speed Ranking (fastest to slowest)
1. **zerolog** - Consistently the fastest across all scenarios
2. **zap** - Excellent performance, especially with fields
3. **slog** - Good performance from standard library
4. **golog** - Competitive with logrus, focuses on features over raw speed
5. **logrus** - Slowest of the tested libraries

### Memory Allocation Ranking (lowest to highest)
1. **zerolog** - Zero allocations in most scenarios
2. **zap** - Minimal allocations (0-1 per log)
3. **slog** - Low allocations (0-9 per log depending on scenario)
4. **golog** - Consistent ~1050 B/op with 2 allocs/op
5. **logrus** - Highest allocations (882-3996 B/op)

### Key Observations

**zerolog's Zero-Allocation Design**
- Uses a fluent API with sync.Pool for zero allocations
- Optimized for high-throughput scenarios
- Fastest in all tested scenarios

**zap's Performance**
- Excellent balance of speed and features
- More allocations than zerolog but still minimal
- Strong performance with complex fields

**slog's Standard Library Benefits**
- No external dependencies
- Good performance for a standard library solution
- Built-in support in Go 1.21+

**golog's Design Trade-offs**
- Focuses on flexibility and feature richness
- Consistent memory profile across scenarios
- Better than logrus, competitive with slog in some scenarios
- Excellent disabled logging performance (79 ns/op, 0 allocs)

**logrus Performance**
- Mature library with wide adoption
- Slower and more allocations than modern alternatives
- Consider migrating to newer libraries for performance-critical applications

## When to Choose Each Library

**Choose zerolog when:**
- Raw performance is critical
- High-throughput logging is needed
- Minimizing allocations is important

**Choose zap when:**
- You need excellent performance with rich features
- Production logging at scale
- Strong typing is important

**Choose slog when:**
- You prefer standard library solutions
- Go 1.21+ is available
- Good-enough performance with no dependencies

**Choose golog when:**
- You need flexible configuration options
- Multiple output formats and writers are required
- Feature richness is more important than raw speed
- Integration with existing domonda ecosystem

**Choose logrus when:**
- You have existing code using it
- Performance is not a primary concern
- Wide plugin ecosystem is needed

## Optimization Opportunities

Based on these benchmarks, potential optimization areas for golog:

1. **Memory pooling**: Consider sync.Pool for message/buffer reuse
2. **Zero-allocation paths**: Optimize hot paths to reduce allocations
3. **Buffer pre-allocation**: Size buffers based on typical log sizes
4. **Inlining**: Ensure critical methods are inlined by the compiler

See [ALLOCATIONS.md](ALLOCATIONS.md) for a detailed analysis of where allocations occur and specific optimization recommendations.

## References

- [Better Stack Go Logging Benchmarks](https://betterstack.com/community/guides/logging/best-golang-logging-libraries/)
- [Better Stack Go Logging Benchmarks Repository](https://github.com/betterstack-community/go-logging-benchmarks)
- [Zerolog Repository](https://github.com/rs/zerolog)
- [Zap Repository](https://github.com/uber-go/zap)
- [Slog Documentation](https://pkg.go.dev/log/slog)
- [Logrus Repository](https://github.com/sirupsen/logrus)
