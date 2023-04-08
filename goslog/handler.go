package goslog

import (
	"context"

	"golang.org/x/exp/slog"

	"github.com/domonda/golog"
)

type ConvertLevelFunc func(slog.Level) golog.Level

func ConvertDefaultLevels(l slog.Level) golog.Level {
	var i int
	switch {
	case l <= slog.LevelDebug:
		i = int(golog.DefaultLevels.Debug) - int(slog.LevelDebug-l)
	case l <= slog.LevelInfo:
		i = int(golog.DefaultLevels.Info) - int(slog.LevelInfo-l)
	case l <= slog.LevelWarn:
		i = int(golog.DefaultLevels.Warn) - int(slog.LevelWarn-l)
	case l <= slog.LevelError:
		i = int(golog.DefaultLevels.Error) - int(slog.LevelError-l)
	default:
		i = int(golog.DefaultLevels.Error) + int(l-slog.LevelError)
	}
	if i < int(golog.LevelMin) || i > int(golog.LevelMax) {
		return golog.LevelInvalid
	}
	return golog.Level(i)
}

func Handler(logger *golog.Logger, convertLevel ConvertLevelFunc) slog.Handler {
	return &handler{logger: logger, convertLevel: convertLevel}
}

type handler struct {
	logger       *golog.Logger
	convertLevel ConvertLevelFunc
	attrs        []slog.Attr
	groupPrefix  string
}

func (h *handler) clone() *handler {
	return &handler{
		logger:       h.logger,
		convertLevel: h.convertLevel,
		attrs:        h.attrs,
		groupPrefix:  h.groupPrefix,
	}
}

func (h *handler) Enabled(_ context.Context, level slog.Level) bool {
	return h.logger.IsActive(h.convertLevel(level))
}

func (h *handler) Handle(_ context.Context, r slog.Record) error {
	msg := h.logger.NewMessageAt(r.Time, h.convertLevel(r.Level), r.Message)
	for _, a := range h.attrs {
		msg = writeAttr(msg, h.groupPrefix, a.Key, a.Value)
	}
	msg.Log()
	return nil
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	with := h.clone()
	if with.attrs == nil {
		with.attrs = attrs
	} else {
		with.attrs = append(with.attrs, attrs...)
	}
	return with
}

func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	with := h.clone()
	with.groupPrefix = prefix(with.groupPrefix, name)
	return with
}

func writeAttr(m *golog.Message, group, key string, value slog.Value) *golog.Message {
	if key == "" {
		return m
	}
	switch value.Kind() {
	case slog.KindAny:
		return m.Any(prefix(group, key), value.Any())
	case slog.KindBool:
		return m.Bool(prefix(group, key), value.Bool())
	case slog.KindDuration:
		return m.Duration(prefix(group, key), value.Duration())
	case slog.KindFloat64:
		return m.Float(prefix(group, key), value.Float64())
	case slog.KindInt64:
		return m.Int64(prefix(group, key), value.Int64())
	case slog.KindString:
		return m.Str(prefix(group, key), value.String())
	case slog.KindTime:
		return m.Time(prefix(group, key), value.Time())
	case slog.KindUint64:
		return m.Uint64(prefix(group, key), value.Uint64())
	case slog.KindGroup:
		for _, attr := range value.Group() {
			m = writeAttr(m, prefix(group, key), attr.Key, attr.Value)
		}
		return m
	case slog.KindLogValuer:
		return writeAttr(m, group, key, value.LogValuer().LogValue())
	default:
		// Should never happen, but don't panic and still log invalid value
		return m.Any(prefix(group, key), value)
	}
}

func prefix(current, name string) string {
	if current == "" {
		return name
	}
	if name == "" {
		return current
	}
	return current + "." + name
}
