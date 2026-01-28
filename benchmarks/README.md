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

BenchmarkSimpleMessage/golog-8         	 9047852	       243 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/zerolog-8       	47294134	        53 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/zap-8           	10576071	       226 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/slog-8          	 6395048	       397 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/logrus-8        	 2768839	       867 ns/op	     889 B/op	      20 allocs/op

BenchmarkWithFields/golog-8            	 6909861	       351 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithFields/zerolog-8          	15290960	       164 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithFields/zap-8              	 5128024	       461 ns/op	     256 B/op	       1 allocs/op
BenchmarkWithFields/slog-8             	 2815092	       851 ns/op	     208 B/op	       9 allocs/op
BenchmarkWithFields/logrus-8           	 1319150	      1823 ns/op	    1938 B/op	      34 allocs/op

BenchmarkWithManyFields/golog-8        	 3837804	       538 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithManyFields/zerolog-8      	 9630970	       253 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithManyFields/zap-8          	 3188349	       746 ns/op	     704 B/op	       1 allocs/op
BenchmarkWithManyFields/slog-8         	 1814827	      1317 ns/op	     448 B/op	       7 allocs/op
BenchmarkWithManyFields/logrus-8       	  679930	      3794 ns/op	    3998 B/op	      53 allocs/op

BenchmarkWithAccumulatedContext/golog-8         	 5823658	       413 ns/op	      24 B/op	       1 allocs/op
BenchmarkWithAccumulatedContext/zerolog-8       	27226228	        84 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithAccumulatedContext/zap-8           	 6943051	       343 ns/op	     128 B/op	       1 allocs/op
BenchmarkWithAccumulatedContext/slog-8          	 4208593	       604 ns/op	      48 B/op	       3 allocs/op
BenchmarkWithAccumulatedContext/logrus-8        	 1232407	      1941 ns/op	    1882 B/op	      32 allocs/op

BenchmarkDisabled/golog-8              	58892754	        47 ns/op	       0 B/op	       0 allocs/op
BenchmarkDisabled/zerolog-8            	360339416	         6 ns/op	       0 B/op	       0 allocs/op
BenchmarkDisabled/zap-8                	63818754	        42 ns/op	     128 B/op	       1 allocs/op
BenchmarkDisabled/slog-8               	52656068	        49 ns/op	      48 B/op	       3 allocs/op
BenchmarkDisabled/logrus-8             	 8137533	       279 ns/op	     528 B/op	       6 allocs/op

BenchmarkComplexFields/golog-8         	 6631597	       369 ns/op	       0 B/op	       0 allocs/op
BenchmarkComplexFields/zerolog-8       	17157057	       146 ns/op	       0 B/op	       0 allocs/op
BenchmarkComplexFields/zap-8           	 4746888	       532 ns/op	     256 B/op	       1 allocs/op
BenchmarkComplexFields/slog-8          	 2613343	       875 ns/op	     104 B/op	       6 allocs/op
BenchmarkComplexFields/logrus-8        	 1000000	      2350 ns/op	    2022 B/op	      36 allocs/op

BenchmarkTextOutput/golog-8            	 2813985	       888 ns/op	     322 B/op	       7 allocs/op
BenchmarkTextOutput/zerolog-8          	  985879	      2745 ns/op	    1849 B/op	      46 allocs/op
BenchmarkTextOutput/zap-8              	 2878914	       731 ns/op	     192 B/op	       4 allocs/op
BenchmarkTextOutput/slog-8             	 3564046	       677 ns/op	      48 B/op	       3 allocs/op
BenchmarkTextOutput/logrus-8           	 1571800	      1580 ns/op	    1245 B/op	      21 allocs/op
```

### Golog Allocation Summary

| Scenario                       | ns/op   | B/op  | allocs/op |
|--------------------------------|---------|-------|-----------|
| Simple Message                 | 243     | 0     | 0         |
| With Fields (4 fields)         | 351     | 0     | 0         |
| Many Fields (10 fields)        | 538     | 0     | 0         |
| Accumulated Context            | 413     | 24    | 1         |
| Complex Fields (error, time)   | 369     | 0     | 0         |
| Text Output                    | 888     | 322   | 7         |
| **Disabled Logging**           | **47**  | **0** | **0**     |

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
- **Text Output**: Human-readable formatting requires additional string allocations

**Note on zero allocations for Complex Fields:**
With the native `Time` attrib type, golog now achieves zero allocations even when logging `error` and `time.Time` fields, matching zerolog's allocation-free design for complex field types.

## Performance Analysis

### JSON Output Speed Ranking (fastest to slowest)
1. **zerolog** - Fastest for JSON output across all scenarios (53-253 ns/op)
2. **golog** - Excellent performance, faster than zap with fields (243-538 ns/op)
3. **zap** - Strong performance (226-746 ns/op)
4. **slog** - Good performance from standard library (397-1317 ns/op)
5. **logrus** - Slowest of the tested libraries (867-3794 ns/op)

### Text/Console Output Speed Ranking (fastest to slowest)
1. **slog** - Fastest for text output (677 ns/op)
2. **zap** - Strong text formatting performance (731 ns/op)
3. **golog** - Competitive text output (888 ns/op)
4. **logrus** - Moderate performance (1580 ns/op)
5. **zerolog** - ConsoleWriter is significantly slower (2745 ns/op, 46 allocs)

### Memory Allocation Ranking (lowest to highest)
1. **zerolog** - Zero allocations for JSON output
2. **golog** - Zero allocations for JSON logging (0 B/op, 0 allocs/op)
3. **zap** - Minimal allocations (0-1 per log)
4. **slog** - Variable allocations (0-9 per log depending on scenario)
5. **logrus** - Highest allocations (889-3998 B/op)

### Key Observations

**zerolog's Zero-Allocation Design**
- Uses a fluent API with sync.Pool for zero allocations in JSON mode
- Optimized for high-throughput JSON logging scenarios
- Fastest for JSON output (49-262 ns/op)
- **Important**: ConsoleWriter for text output is significantly slower (2365 ns/op) with 46 allocations per log entry—this is a known limitation of zerolog's text formatting

**zap's Performance**
- Excellent balance of speed and features
- More allocations than zerolog but still minimal
- Strong performance with complex fields (260-647 ns/op)
- **Best choice for text/console output** (455 ns/op)

**golog's Zero-Allocation Design**
- Zero allocations for all JSON logging scenarios including complex fields (0 B/op, 0 allocs/op)
- Native `Time` and `Times` attrib types eliminate allocations for time.Time logging
- Faster than slog for JSON: 243 vs 397 ns/op (simple), 351 vs 851 ns/op (with fields)
- Faster than zap with fields: 351 vs 461 ns/op (4 fields), 538 vs 746 ns/op (10 fields)
- Complex fields (error, time): 369 ns/op with 0 allocs vs zap's 532 ns/op with 1 alloc
- Excellent disabled logging performance (47 ns/op, 0 allocs)
- Uses embedded array in Message struct for writers slice
- Competitive text output performance (888 ns/op)—faster than zerolog's ConsoleWriter
- Flexible multi-writer architecture

**slog's Standard Library Benefits**
- No external dependencies
- Good performance for a standard library solution
- Built-in support in Go 1.21+
- Solid text output performance (625 ns/op)

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
