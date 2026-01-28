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

## Benchmark Results

Results from Apple M2:

```
goos: darwin
goarch: arm64
cpu: Apple M2

BenchmarkSimpleMessage/golog-8         	 3447380	       329 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/zerolog-8       	25235973	        52 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/zap-8           	 5395564	       221 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/slog-8          	 3345938	       369 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/logrus-8        	 1404438	       856 ns/op	     889 B/op	      20 allocs/op

BenchmarkWithFields/golog-8            	 2112727	       568 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithFields/zerolog-8          	 8752854	       138 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithFields/zap-8              	 2934567	       476 ns/op	     256 B/op	       1 allocs/op
BenchmarkWithFields/slog-8             	 1453699	       829 ns/op	     208 B/op	       9 allocs/op
BenchmarkWithFields/logrus-8           	  633794	      1806 ns/op	    1938 B/op	      34 allocs/op

BenchmarkWithManyFields/golog-8        	 1412446	       811 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithManyFields/zerolog-8      	 4941500	       243 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithManyFields/zap-8          	 1785013	       665 ns/op	     704 B/op	       1 allocs/op
BenchmarkWithManyFields/slog-8         	  939159	      1432 ns/op	     448 B/op	       7 allocs/op
BenchmarkWithManyFields/logrus-8       	  337508	      3480 ns/op	    3998 B/op	      53 allocs/op

BenchmarkWithAccumulatedContext/golog-8         	 1641970	       731 ns/op	      24 B/op	       1 allocs/op
BenchmarkWithAccumulatedContext/zerolog-8       	14541458	        85 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithAccumulatedContext/zap-8           	 3789568	       317 ns/op	     128 B/op	       1 allocs/op
BenchmarkWithAccumulatedContext/slog-8          	 2196698	       548 ns/op	      48 B/op	       3 allocs/op
BenchmarkWithAccumulatedContext/logrus-8        	  672468	      1812 ns/op	    1882 B/op	      32 allocs/op

BenchmarkDisabled/golog-8              	29177704	        45 ns/op	       0 B/op	       0 allocs/op
BenchmarkDisabled/zerolog-8            	308841686	         4 ns/op	       0 B/op	       0 allocs/op
BenchmarkDisabled/zap-8                	43230130	        27 ns/op	     128 B/op	       1 allocs/op
BenchmarkDisabled/slog-8               	28879158	        41 ns/op	      48 B/op	       3 allocs/op
BenchmarkDisabled/logrus-8             	 4934780	       241 ns/op	     528 B/op	       6 allocs/op

BenchmarkComplexFields/golog-8         	 1237047	       979 ns/op	     128 B/op	       3 allocs/op
BenchmarkComplexFields/zerolog-8       	 9128604	       133 ns/op	       0 B/op	       0 allocs/op
BenchmarkComplexFields/zap-8           	 2654292	       466 ns/op	     256 B/op	       1 allocs/op
BenchmarkComplexFields/slog-8          	 1519286	       782 ns/op	     104 B/op	       6 allocs/op
BenchmarkComplexFields/logrus-8        	  583484	      2030 ns/op	    2022 B/op	      36 allocs/op

BenchmarkTextOutput/golog-8            	 1515180	       794 ns/op	     322 B/op	       7 allocs/op
BenchmarkTextOutput/zerolog-8          	15130888	        79 ns/op	       0 B/op	       0 allocs/op
BenchmarkTextOutput/zap-8              	 2640254	       498 ns/op	     192 B/op	       4 allocs/op
BenchmarkTextOutput/slog-8             	 1950204	       618 ns/op	      48 B/op	       3 allocs/op
BenchmarkTextOutput/logrus-8           	  905787	      1309 ns/op	    1245 B/op	      21 allocs/op
```

### Golog Allocation Summary

| Scenario                       | ns/op  | B/op  | allocs/op |
|--------------------------------|--------|-------|-----------|
| Simple Message                 | 329    | 0     | 0         |
| With Fields (4 fields)         | 568    | 0     | 0         |
| Many Fields (10 fields)        | 811    | 0     | 0         |
| Accumulated Context            | 731    | 24    | 1         |
| Complex Fields (error, time)   | 979    | 128   | 3         |
| Text Output                    | 794    | 322   | 7         |
| **Disabled Logging**           | **45** | **0** | **0**     |

### Library Allocation Comparison

| Library     | Simple Message      | With Fields          | Disabled           |
|-------------|---------------------|----------------------|--------------------|
| **zerolog** | 0 B/op, 0 allocs    | 0 B/op, 0 allocs     | 0 B/op, 0 allocs   |
| **golog**   | 0 B/op, 0 allocs    | 0 B/op, 0 allocs     | 0 B/op, 0 allocs   |
| **zap**     | 0 B/op, 0 allocs    | 256 B/op, 1 alloc    | 128 B/op, 1 alloc  |
| **slog**    | 0 B/op, 0 allocs    | 208 B/op, 9 allocs   | 48 B/op, 3 allocs  |
| **logrus**  | 889 B/op, 20 allocs | 1939 B/op, 34 allocs | 528 B/op, 6 allocs |

**Note on remaining allocations:**
- **Accumulated Context**: Sub-logger creation requires cloning the attribs slice
- **Complex Fields**: Error and time formatting (`error.Error()`, `time.Time.String()`) allocate
- **Text Output**: Human-readable formatting requires additional string allocations

## Performance Analysis

### Speed Ranking (fastest to slowest)
1. **zerolog** - Consistently the fastest across all scenarios
2. **zap** - Excellent performance, especially with fields
3. **slog** - Good performance from standard library
4. **golog** - Competitive with slog, much better than logrus
5. **logrus** - Slowest of the tested libraries

### Memory Allocation Ranking (lowest to highest)
1. **zerolog** - Zero allocations in most scenarios
2. **golog** - Zero allocations for standard logging (0 B/op, 0 allocs/op)
3. **zap** - Minimal allocations (0-1 per log)
4. **slog** - Variable allocations (0-9 per log depending on scenario)
5. **logrus** - Highest allocations (889-3998 B/op)

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

**golog's Zero-Allocation Design**
- Zero allocations for standard JSON logging (0 B/op, 0 allocs/op)
- Excellent disabled logging performance (45 ns/op, 0 allocs)
- Uses embedded array in Message struct for writers slice
- Some allocations for text output and complex fields (error formatting)
- Flexible multi-writer architecture

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
- Zero allocations when disabled is important
- Integration with existing domonda ecosystem

**Choose logrus when:**
- You have existing code using it
- Performance is not a primary concern
- Wide plugin ecosystem is needed

## References

- [Better Stack Go Logging Benchmarks](https://betterstack.com/community/guides/logging/best-golang-logging-libraries/)
- [Better Stack Go Logging Benchmarks Repository](https://github.com/betterstack-community/go-logging-benchmarks)
- [Zerolog Repository](https://github.com/rs/zerolog)
- [Zap Repository](https://github.com/uber-go/zap)
- [Slog Documentation](https://pkg.go.dev/log/slog)
- [Logrus Repository](https://github.com/sirupsen/logrus)
