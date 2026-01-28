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

BenchmarkSimpleMessage/golog-8         	 1882119	       638 ns/op	      24 B/op	       1 allocs/op
BenchmarkSimpleMessage/zerolog-8       	14454950	        83 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/zap-8           	 3103516	       388 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/slog-8          	 1939730	       627 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleMessage/logrus-8        	  775527	      1695 ns/op	     889 B/op	      20 allocs/op

BenchmarkWithFields/golog-8            	 1000000	      1066 ns/op	      24 B/op	       1 allocs/op
BenchmarkWithFields/zerolog-8          	 4985748	       240 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithFields/zap-8              	 1726173	       698 ns/op	     256 B/op	       1 allocs/op
BenchmarkWithFields/slog-8             	  853892	      1422 ns/op	     208 B/op	       9 allocs/op
BenchmarkWithFields/logrus-8           	  376412	      3066 ns/op	    1939 B/op	      34 allocs/op

BenchmarkWithManyFields/golog-8        	  780331	      1741 ns/op	      24 B/op	       1 allocs/op
BenchmarkWithManyFields/zerolog-8      	 2786185	       472 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithManyFields/zap-8          	 1000000	      1133 ns/op	     704 B/op	       1 allocs/op
BenchmarkWithManyFields/slog-8         	  541113	      2311 ns/op	     448 B/op	       7 allocs/op
BenchmarkWithManyFields/logrus-8       	  199092	      6057 ns/op	    3998 B/op	      53 allocs/op

BenchmarkWithAccumulatedContext/golog-8         	  916940	      1319 ns/op	      48 B/op	       2 allocs/op
BenchmarkWithAccumulatedContext/zerolog-8       	 8606998	       140 ns/op	       0 B/op	       0 allocs/op
BenchmarkWithAccumulatedContext/zap-8           	 2105499	       545 ns/op	     128 B/op	       1 allocs/op
BenchmarkWithAccumulatedContext/slog-8          	 1000000	      1058 ns/op	      48 B/op	       3 allocs/op
BenchmarkWithAccumulatedContext/logrus-8        	  385596	      3065 ns/op	    1882 B/op	      32 allocs/op

BenchmarkDisabled/golog-8              	17221944	        70 ns/op	       0 B/op	       0 allocs/op
BenchmarkDisabled/zerolog-8            	180251212	         7 ns/op	       0 B/op	       0 allocs/op
BenchmarkDisabled/zap-8                	27230875	        44 ns/op	     128 B/op	       1 allocs/op
BenchmarkDisabled/slog-8               	17217342	        69 ns/op	      48 B/op	       3 allocs/op
BenchmarkDisabled/logrus-8             	 3024298	       395 ns/op	     528 B/op	       6 allocs/op

BenchmarkComplexFields/golog-8         	  697248	      1848 ns/op	     152 B/op	       4 allocs/op
BenchmarkComplexFields/zerolog-8       	 5323473	       225 ns/op	       0 B/op	       0 allocs/op
BenchmarkComplexFields/zap-8           	 1561759	       763 ns/op	     256 B/op	       1 allocs/op
BenchmarkComplexFields/slog-8          	  904917	      1333 ns/op	     104 B/op	       6 allocs/op
BenchmarkComplexFields/logrus-8        	  347187	      3891 ns/op	    2022 B/op	      36 allocs/op

BenchmarkTextOutput/golog-8            	  859528	      1550 ns/op	     347 B/op	       8 allocs/op
BenchmarkTextOutput/zerolog-8          	 7879134	       135 ns/op	       0 B/op	       0 allocs/op
BenchmarkTextOutput/zap-8              	 1322112	       835 ns/op	     192 B/op	       4 allocs/op
BenchmarkTextOutput/slog-8             	 1000000	      1056 ns/op	      48 B/op	       3 allocs/op
BenchmarkTextOutput/logrus-8           	  511267	      2251 ns/op	    1245 B/op	      21 allocs/op
```

## Performance Analysis

### Speed Ranking (fastest to slowest)
1. **zerolog** - Consistently the fastest across all scenarios
2. **zap** - Excellent performance, especially with fields
3. **slog** - Good performance from standard library
4. **golog** - Competitive with slog, much better than logrus
5. **logrus** - Slowest of the tested libraries

### Memory Allocation Ranking (lowest to highest)
1. **zerolog** - Zero allocations in most scenarios
2. **zap** - Minimal allocations (0-1 per log)
3. **golog** - Consistent 24 B/op with 1 alloc/op for JSON output
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

**golog's Design Trade-offs**
- Consistent low memory profile (24 B/op) across most scenarios
- Excellent disabled logging performance (70 ns/op, 0 allocs)
- More allocations for text output and complex fields
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

- [ALLOCATIONS.md](ALLOCATIONS.md) - Detailed analysis of where allocations occur
- [Better Stack Go Logging Benchmarks](https://betterstack.com/community/guides/logging/best-golang-logging-libraries/)
- [Better Stack Go Logging Benchmarks Repository](https://github.com/betterstack-community/go-logging-benchmarks)
- [Zerolog Repository](https://github.com/rs/zerolog)
- [Zap Repository](https://github.com/uber-go/zap)
- [Slog Documentation](https://pkg.go.dev/log/slog)
- [Logrus Repository](https://github.com/sirupsen/logrus)
