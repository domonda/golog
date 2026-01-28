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
go test ./benchmarks -bench=. -benchmem -benchtime=2s
```

Or from the benchmarks directory:

```bash
cd benchmarks
go test -bench=. -benchmem -benchtime=2s
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

```text
goos: darwin
goarch: arm64
cpu: Apple M2

BenchmarkSimpleMessage/golog-8         	 8545741	       250 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/zerolog-8       	47107233	        51 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/zap-8           	 9502837	       248 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/slog-8          	 5645415	       389 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/logrus-8        	 2653404	       883 ns/op	     889 B/op	      20 allocs/op

BenchmarkWithFields/golog-8            	 6370593	       353 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithFields/zerolog-8          	16187962	       145 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithFields/zap-8              	 5135613	       451 ns/op	     256 B/op	       1 allocs/op
BenchmarkWithFields/slog-8             	 2710246	       860 ns/op	     208 B/op	       9 allocs/op
BenchmarkWithFields/logrus-8           	 1310167	      1902 ns/op	    1939 B/op	      34 allocs/op

BenchmarkWithManyFields/golog-8        	 4682881	       493 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithManyFields/zerolog-8      	 8832258	       249 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithManyFields/zap-8          	 3237511	       741 ns/op	     704 B/op	       1 allocs/op
BenchmarkWithManyFields/slog-8         	 1839981	      1366 ns/op	     448 B/op	       7 allocs/op
BenchmarkWithManyFields/logrus-8       	  601110	      3861 ns/op	    3998 B/op	      53 allocs/op

BenchmarkWithAccumulatedContext/golog-8         	 5554165	       411 ns/op	      24 B/op	       1 allocs/op
BenchmarkWithAccumulatedContext/zerolog-8       	27960081	        85 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithAccumulatedContext/zap-8           	 6807020	       353 ns/op	     128 B/op	       1 allocs/op
BenchmarkWithAccumulatedContext/slog-8          	 4073331	       575 ns/op	      48 B/op	       3 allocs/op
BenchmarkWithAccumulatedContext/logrus-8        	 1337250	      1779 ns/op	    1882 B/op	      32 allocs/op

BenchmarkDisabled/golog-8              	56760769	        42 ns/op	       0 B/op	       0 allocs/op
BenchmarkDisabled/zerolog-8            	423263750	         6 ns/op	       0 B/op	       0 allocs/op
BenchmarkDisabled/zap-8                	68156506	        35 ns/op	     128 B/op	       1 allocs/op
BenchmarkDisabled/slog-8               	50444190	        47 ns/op	      48 B/op	       3 allocs/op
BenchmarkDisabled/logrus-8             	 8593213	       250 ns/op	     528 B/op	       6 allocs/op

BenchmarkComplexFields/golog-8         	 6241108	       391 ns/op	       0 B/op	       0 allocs/op
BenchmarkComplexFields/zerolog-8       	17020742	       134 ns/op	       0 B/op	       0 allocs/op
BenchmarkComplexFields/zap-8           	 4860870	       475 ns/op	     256 B/op	       1 allocs/op
BenchmarkComplexFields/slog-8          	 2878185	       816 ns/op	     104 B/op	       6 allocs/op
BenchmarkComplexFields/logrus-8        	 1055866	      2216 ns/op	    2022 B/op	      36 allocs/op

BenchmarkTextOutput/golog-8            	 5618756	       394 ns/op	       0 B/op	       0 allocs/op
BenchmarkTextOutput/zerolog-8          	  990175	      2363 ns/op	    1849 B/op	      46 allocs/op
BenchmarkTextOutput/zap-8              	 4499472	       514 ns/op	     192 B/op	       4 allocs/op
BenchmarkTextOutput/slog-8             	 3716494	       677 ns/op	      48 B/op	       3 allocs/op
BenchmarkTextOutput/logrus-8           	 1784671	      1338 ns/op	    1245 B/op	      21 allocs/op
```

### Golog Allocation Summary

| Scenario                       | ns/op   | B/op  | allocs/op |
|--------------------------------|---------|-------|-----------|
| Simple Message                 | 250     | 0     | 0         |
| With Fields (4 fields)         | 353     | 0     | 0         |
| Many Fields (10 fields)        | 493     | 0     | 0         |
| Accumulated Context            | 411     | 24    | 1         |
| Complex Fields (error, time)   | 391     | 0     | 0         |
| Text Output                    | 394     | 0     | 0         |
| **Disabled Logging**           | **42**  | **0** | **0**     |

### Library Allocation Comparison

| Library     | Simple Message      | With Fields          | Disabled           |
|-------------|---------------------|----------------------|--------------------|
| **zerolog** | 0 B/op, 0 allocs    | 0 B/op, 0 allocs     | 0 B/op, 0 allocs   |
| **golog**   | 0 B/op, 0 allocs    | 0 B/op, 0 allocs     | 0 B/op, 0 allocs   |
| **zap**     | 0 B/op, 0 allocs    | 256 B/op, 1 alloc    | 128 B/op, 1 alloc  |
| **slog**    | 0 B/op, 0 allocs    | 208 B/op, 9 allocs   | 48 B/op, 3 allocs  |
| **logrus**  | 889 B/op, 20 allocs | 1939 B/op, 34 allocs | 528 B/op, 6 allocs |

**Note on remaining allocation:**
- **Accumulated Context**: Sub-logger creation requires cloning the attribs slice

**Note on zero allocations for Complex Fields:**
With the native `Time` attrib type, golog now achieves zero allocations even when logging `error` and `time.Time` fields, matching zerolog's allocation-free design for complex field types.

## Performance Analysis

### JSON Output Speed Ranking (fastest to slowest)
1. **zerolog** - Fastest for JSON output across all scenarios (51-249 ns/op)
2. **golog** - Excellent performance, faster than zap with fields (250-493 ns/op)
3. **zap** - Strong performance (248-741 ns/op)
4. **slog** - Good performance from standard library (389-1366 ns/op)
5. **logrus** - Slowest of the tested libraries (883-3861 ns/op)

### Text/Console Output Speed Ranking (fastest to slowest)
1. **golog** - Fastest for text output (394 ns/op, 0 allocs)
2. **zap** - Strong text formatting performance (514 ns/op)
3. **slog** - Good text formatting (677 ns/op)
4. **logrus** - Moderate performance (1338 ns/op)
5. **zerolog** - ConsoleWriter is significantly slower (2363 ns/op, 46 allocs)

### Memory Allocation Ranking (lowest to highest)
1. **zerolog** - Zero allocations for JSON output
2. **golog** - Zero allocations for JSON and text logging (0 B/op, 0 allocs/op)
3. **zap** - Minimal allocations (0-1 per log)
4. **slog** - Variable allocations (0-9 per log depending on scenario)
5. **logrus** - Highest allocations (889-3998 B/op)

### Key Observations

**zerolog's Zero-Allocation Design**
- Uses a fluent API with sync.Pool for zero allocations in JSON mode
- Optimized for high-throughput JSON logging scenarios
- Fastest for JSON output (51-249 ns/op)
- **Important**: ConsoleWriter for text output is significantly slower (2363 ns/op) with 46 allocations per log entry—this is a known limitation of zerolog's text formatting

**zap's Performance**
- Excellent balance of speed and features
- More allocations than zerolog but still minimal
- Strong performance with complex fields (248-741 ns/op)
- Good text/console output performance (514 ns/op)

**golog's Zero-Allocation Design**
- Zero allocations for all logging scenarios: JSON, text, and complex fields (0 B/op, 0 allocs/op)
- Faster than slog for JSON: 250 vs 389 ns/op (simple), 353 vs 860 ns/op (with fields)
- Faster than zap with fields: 353 vs 451 ns/op (4 fields), 493 vs 741 ns/op (10 fields)
- Complex fields (error, time): 391 ns/op with 0 allocs vs zap's 475 ns/op with 1 alloc
- Excellent disabled logging performance (42 ns/op, 0 allocs)
- **Fastest text output** (394 ns/op, 0 allocs) with append-style formatting—faster than zap (514 ns/op) and slog (677 ns/op)
- Flexible multi-writer architecture

**slog's Standard Library Benefits**
- No external dependencies
- Good performance for a standard library solution
- Built-in support in Go 1.21+
- Solid text output performance (677 ns/op)

**logrus Performance**
- Mature library with wide adoption
- Slower and more allocations than modern alternatives
- Consider migrating to newer libraries for performance-critical applications

## When to Choose Each Library

**Choose golog when:**
- You need flexible configuration options
- Multiple output formats and writers are required
- Zero allocations when disabled is important
- Integration with existing domonda ecosystem

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
