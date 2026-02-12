package otel

import (
	"context"
	"fmt"
	"math"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/log"

	"github.com/domonda/golog"
)

var (
	_ golog.Writer       = new(Writer)
	_ golog.WriterConfig = new(WriterConfig)
)

// WriterConfig implements golog.WriterConfig and serves as a factory for
// creating Writer instances that emit log records to OpenTelemetry.
//
// Example usage:
//
//	config := otel.NewWriterConfig(
//	    loggerProvider,
//	    golog.NewDefaultFormat(),
//	    golog.AllActive,
//	    log.String("service", "my-app"),
//	)
type WriterConfig struct {
	logger     log.Logger
	format     *golog.Format
	filter     golog.LevelFilter
	attrs      []log.KeyValue
	writerPool sync.Pool
}

// NewWriterConfig returns a new WriterConfig for an OpenTelemetry LoggerProvider.
// The provider is used to create a Logger with the instrumentation name
// "github.com/domonda/golog/otel".
// Any KeyValue pairs passed as attrs will be added to every log record.
func NewWriterConfig(provider log.LoggerProvider, format *golog.Format, filter golog.LevelFilter, attrs ...log.KeyValue) *WriterConfig {
	return &WriterConfig{
		logger: provider.Logger("github.com/domonda/golog/otel"),
		format: format,
		filter: filter,
		attrs:  attrs,
	}
}

func (c *WriterConfig) WriterForNewMessage(ctx context.Context, level golog.Level) golog.Writer {
	if c.filter.IsInactive(ctx, level) || IsContextWithoutLogging(ctx) {
		return nil
	}
	if w, ok := c.writerPool.Get().(*Writer); ok && w != nil {
		w.ctx = ctx
		return w
	}
	return &Writer{config: c, ctx: ctx}
}

// FlushUnderlying is a no-op because the OTel Log API interface
// (log.LoggerProvider) does not expose Flush or Shutdown methods.
// The caller must flush the SDK-level provider directly, e.g. via
// sdklog.LoggerProvider.ForceFlush or sdklog.LoggerProvider.Shutdown.
func (c *WriterConfig) FlushUnderlying() {}

///////////////////////////////////////////////////////////////////////////////

// Writer implements golog.Writer and handles the actual emission of log
// records to OpenTelemetry. It accumulates log data during the logging process
// and emits it as an OTel log record when CommitMessage() is called.
//
// Writer instances are reused through object pooling to minimize allocations.
//
// The Writer automatically maps golog levels to OTel severities:
//
//	TRACE -> SeverityTrace, DEBUG -> SeverityDebug, INFO -> SeverityInfo,
//	WARN -> SeverityWarn, ERROR -> SeverityError, FATAL -> SeverityFatal
type Writer struct {
	config       *WriterConfig
	ctx          context.Context
	timestamp    time.Time
	severity     log.Severity
	severityText string
	message      strings.Builder
	attrs        []log.KeyValue
	key          string
	sliceKey     string
	sliceValues  []log.Value
}

func (w *Writer) BeginMessage(config golog.Config, timestamp time.Time, level golog.Level, prefix, text string) {
	w.timestamp = timestamp

	levels := config.Levels()
	switch level {
	case levels.Fatal:
		w.severity = log.SeverityFatal
		w.severityText = "FATAL"
	case levels.Error:
		w.severity = log.SeverityError
		w.severityText = "ERROR"
	case levels.Warn:
		w.severity = log.SeverityWarn
		w.severityText = "WARN"
	case levels.Info:
		w.severity = log.SeverityInfo
		w.severityText = "INFO"
	case levels.Debug:
		w.severity = log.SeverityDebug
		w.severityText = "DEBUG"
	case levels.Trace:
		w.severity = log.SeverityTrace
		w.severityText = "TRACE"
	default:
		w.severity = UnknownSeverity
		w.severityText = ""
	}

	if prefix != "" {
		fmt.Fprintf(&w.message, w.config.format.PrefixFmt, prefix, text)
	} else {
		w.message.WriteString(text)
	}
}

func (w *Writer) CommitMessage() {
	defer func() {
		if r := recover(); r != nil {
			golog.ErrorHandler(fmt.Errorf("otel.Writer.CommitMessage recovered panic: %v\n%s", r, debug.Stack()))
		}

		// Reset and return to pool
		w.message.Reset()
		w.attrs = w.attrs[:0]
		w.sliceValues = nil
		w.sliceKey = ""
		w.key = ""
		w.ctx = nil
		w.config.writerPool.Put(w)
	}()

	if w.message.Len() > 0 {
		var record log.Record
		record.SetTimestamp(w.timestamp)
		record.SetSeverity(w.severity)
		record.SetSeverityText(w.severityText)
		record.SetBody(log.StringValue(w.message.String()))
		record.AddAttributes(w.config.attrs...)
		record.AddAttributes(w.attrs...)

		w.config.logger.Emit(w.ctx, record)
	}
}

func (w *Writer) String() string {
	return w.message.String()
}

func (w *Writer) WriteKey(key string) {
	w.key = key
}

func (w *Writer) WriteSliceKey(key string) {
	w.sliceKey = key
	w.sliceValues = make([]log.Value, 0)
}

func (w *Writer) WriteSliceEnd() {
	w.attrs = append(w.attrs, log.Slice(w.sliceKey, w.sliceValues...))
	w.sliceValues = nil
	w.sliceKey = ""
}

func (w *Writer) WriteNil() {
	w.writeValue(log.Value{})
}

func (w *Writer) WriteBool(val bool) {
	w.writeValue(log.BoolValue(val))
}

func (w *Writer) WriteInt(val int64) {
	w.writeValue(log.Int64Value(val))
}

func (w *Writer) WriteUint(val uint64) {
	if val <= math.MaxInt64 {
		w.writeValue(log.Int64Value(int64(val)))
	} else {
		w.writeValue(log.StringValue(fmt.Sprintf("%d", val)))
	}
}

func (w *Writer) WriteFloat(val float64) {
	w.writeValue(log.Float64Value(val))
}

func (w *Writer) WriteString(val string) {
	w.writeValue(log.StringValue(val))
}

func (w *Writer) WriteError(val error) {
	w.writeValue(log.StringValue(val.Error()))
}

func (w *Writer) WriteTime(val time.Time) {
	w.writeValue(log.StringValue(val.Format(time.RFC3339Nano)))
}

func (w *Writer) WriteUUID(val [16]byte) {
	w.writeValue(log.StringValue(golog.FormatUUID(val)))
}

func (w *Writer) WriteJSON(val []byte) {
	w.writeValue(log.StringValue(string(val)))
}

func (w *Writer) writeValue(val log.Value) {
	if w.sliceValues != nil {
		w.sliceValues = append(w.sliceValues, val)
	} else {
		w.attrs = append(w.attrs, log.KeyValue{Key: w.key, Value: val})
	}
}
