package goslog

import (
	"context"
	"strings"

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

func (h *handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.logger.IsActive(ctx, h.convertLevel(level))
}

func (h *handler) Handle(ctx context.Context, record slog.Record) error {
	msg := h.logger.NewMessageAt(ctx, record.Time, h.convertLevel(record.Level), record.Message)
	for _, a := range h.attrs {
		msg = writeAttr(msg, h.groupPrefix, a.Key, a.Value)
	}
	record.Attrs(func(a slog.Attr) bool {
		msg = writeAttr(msg, h.groupPrefix, a.Key, a.Value)
		return true
	})
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
	with.groupPrefix = prefixKey(with.groupPrefix, name)
	return with
}

func writeAttr(m *golog.Message, group, key string, value slog.Value) *golog.Message {
	kind := value.Kind()
	if kind == slog.KindGroup {
		group = prefixKey(group, key)
		for _, attr := range value.Group() {
			m = writeAttr(m, group, attr.Key, attr.Value)
		}
		return m
	}
	if key == "" {
		return m
	}
	switch kind {
	case slog.KindAny:
		return m.Any(prefixKey(group, key), value.Any())
	case slog.KindBool:
		return m.Bool(prefixKey(group, key), value.Bool())
	case slog.KindDuration:
		return m.Duration(prefixKey(group, key), value.Duration())
	case slog.KindFloat64:
		return m.Float(prefixKey(group, key), value.Float64())
	case slog.KindInt64:
		return m.Int64(prefixKey(group, key), value.Int64())
	case slog.KindString:
		return m.Str(prefixKey(group, key), value.String())
	case slog.KindTime:
		return m.Time(prefixKey(group, key), value.Time())
	case slog.KindUint64:
		return m.Uint64(prefixKey(group, key), value.Uint64())
	case slog.KindLogValuer:
		return writeAttr(m, group, key, value.LogValuer().LogValue().Resolve())
	default:
		// Should never happen, but don't panic and still log any value
		return m.Any(prefixKey(group, key), value)
	}
}

func prefixKey(group, key string) string {
	if group == "" {
		return key
	}
	if key == "" {
		return group
	}
	return group + "." + key
}

func canSplitGroupKey(key string) bool {
	return strings.ContainsRune(key, '.')
}

func splitGroupKeyVal(key string, val any) (rootKey string, rootVal any) {
	if !strings.ContainsRune(key, '.') {
		return key, val
	}

	keys := strings.Split(key, ".")
	groupVals := map[string]any{keys[len(keys)-1]: val}
	for i := len(keys) - 2; i >= 0; i-- {
		groupVals = map[string]any{keys[i]: groupVals}
	}
	return keys[0], groupVals
}
