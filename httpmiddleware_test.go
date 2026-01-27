package golog

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPNoHeaders(t *testing.T) {
	assert.Equal(t, "HTTPNoHeaders", HTTPNoHeaders)
}

func TestGetOrCreateRequestUUID(t *testing.T) {
	t.Run("returns UUID from X-Request-ID header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "a547276f-b02b-4e7d-b67e-c6deb07567da")

		uuid := GetOrCreateRequestUUID(req)

		expected := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")
		assert.Equal(t, expected, uuid)
	})

	t.Run("returns UUID from X-Correlation-ID header when X-Request-ID is missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Correlation-ID", "b657387e-c13c-5f8e-c78f-d7fec18678eb")

		uuid := GetOrCreateRequestUUID(req)

		expected := MustParseUUID("b657387e-c13c-5f8e-c78f-d7fec18678eb")
		assert.Equal(t, expected, uuid)
	})

	t.Run("prefers X-Request-ID over X-Correlation-ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "a547276f-b02b-4e7d-b67e-c6deb07567da")
		req.Header.Set("X-Correlation-ID", "b657387e-c13c-5f8e-c78f-d7fec18678eb")

		uuid := GetOrCreateRequestUUID(req)

		expected := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")
		assert.Equal(t, expected, uuid)
	})

	t.Run("creates new UUID when no header is present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		uuid := GetOrCreateRequestUUID(req)

		// Should return a non-zero UUID
		assert.NotEqual(t, [16]byte{}, uuid)
	})

	t.Run("creates new UUID when header has invalid format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "not-a-valid-uuid")

		uuid := GetOrCreateRequestUUID(req)

		// Should return a non-zero UUID (newly generated)
		assert.NotEqual(t, [16]byte{}, uuid)
	})
}

func TestGetOrCreateRequestID(t *testing.T) {
	t.Run("returns X-Request-ID header value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "custom-request-id-123")

		id := GetOrCreateRequestID(req)

		assert.Equal(t, "custom-request-id-123", id)
	})

	t.Run("returns X-Correlation-ID when X-Request-ID is missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Correlation-ID", "custom-correlation-id-456")

		id := GetOrCreateRequestID(req)

		assert.Equal(t, "custom-correlation-id-456", id)
	})

	t.Run("prefers X-Request-ID over X-Correlation-ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "request-id")
		req.Header.Set("X-Correlation-ID", "correlation-id")

		id := GetOrCreateRequestID(req)

		assert.Equal(t, "request-id", id)
	})

	t.Run("creates new formatted UUID when no header is present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		id := GetOrCreateRequestID(req)

		// Should be a valid UUID string format
		assert.Len(t, id, 36) // UUID string length
		assert.Contains(t, id, "-")
	})

	t.Run("accepts any string value from header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "simple-id")

		id := GetOrCreateRequestID(req)

		assert.Equal(t, "simple-id", id)
	})
}

func TestGetRequestUUIDFromContext(t *testing.T) {
	t.Run("returns UUID from context", func(t *testing.T) {
		ctx := context.Background()
		expectedUUID := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")
		ctx = ContextWithRequestUUID(ctx, expectedUUID)

		uuid, ok := GetRequestUUIDFromContext(ctx)

		assert.True(t, ok)
		assert.Equal(t, expectedUUID, uuid)
	})

	t.Run("returns false when no requestID in context", func(t *testing.T) {
		ctx := context.Background()

		uuid, ok := GetRequestUUIDFromContext(ctx)

		assert.False(t, ok)
		assert.Equal(t, [16]byte{}, uuid)
	})

	t.Run("returns false when requestID is not a UUID type", func(t *testing.T) {
		ctx := context.Background()
		ctx = ContextWithRequestID(ctx, "string-request-id")

		uuid, ok := GetRequestUUIDFromContext(ctx)

		assert.False(t, ok)
		assert.Equal(t, [16]byte{}, uuid)
	})
}

func TestGetRequestIDFromContext(t *testing.T) {
	t.Run("returns string from UUID attribute", func(t *testing.T) {
		ctx := context.Background()
		expectedUUID := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")
		ctx = ContextWithRequestUUID(ctx, expectedUUID)

		id := GetRequestIDFromContext(ctx)

		assert.Equal(t, "a547276f-b02b-4e7d-b67e-c6deb07567da", id)
	})

	t.Run("returns string from string attribute", func(t *testing.T) {
		ctx := context.Background()
		ctx = ContextWithRequestID(ctx, "my-request-id")

		id := GetRequestIDFromContext(ctx)

		assert.Equal(t, "my-request-id", id)
	})

	t.Run("returns empty string when no requestID in context", func(t *testing.T) {
		ctx := context.Background()

		id := GetRequestIDFromContext(ctx)

		assert.Equal(t, "", id)
	})
}

func TestGetOrCreateRequestUUIDFromContext(t *testing.T) {
	t.Run("returns UUID from context", func(t *testing.T) {
		ctx := context.Background()
		expectedUUID := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")
		ctx = ContextWithRequestUUID(ctx, expectedUUID)

		uuid := GetOrCreateRequestUUIDFromContext(ctx)

		assert.Equal(t, expectedUUID, uuid)
	})

	t.Run("creates new UUID when no requestID in context", func(t *testing.T) {
		ctx := context.Background()

		uuid := GetOrCreateRequestUUIDFromContext(ctx)

		// Should return a non-zero UUID
		assert.NotEqual(t, [16]byte{}, uuid)
	})

	t.Run("creates new UUID when requestID is not a UUID type", func(t *testing.T) {
		ctx := context.Background()
		ctx = ContextWithRequestID(ctx, "string-request-id")

		uuid := GetOrCreateRequestUUIDFromContext(ctx)

		// Should return a non-zero UUID
		assert.NotEqual(t, [16]byte{}, uuid)
	})
}

func TestContextWithRequestUUID(t *testing.T) {
	t.Run("adds UUID to context", func(t *testing.T) {
		ctx := context.Background()
		expectedUUID := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")

		ctx = ContextWithRequestUUID(ctx, expectedUUID)

		uuid, ok := GetRequestUUIDFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, expectedUUID, uuid)
	})
}

func TestContextWithRequestID(t *testing.T) {
	t.Run("adds string ID to context", func(t *testing.T) {
		ctx := context.Background()

		ctx = ContextWithRequestID(ctx, "my-request-id")

		id := GetRequestIDFromContext(ctx)
		assert.Equal(t, "my-request-id", id)
	})
}

func TestHTTPMiddlewareHandler(t *testing.T) {
	t.Run("passes request ID to next handler", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, nil)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)
		logger := NewLogger(logConfig)

		var capturedRequestID [16]byte
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedRequestID, _ = GetRequestUUIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		handler := HTTPMiddlewareHandler(nextHandler, logger, DefaultLevels.Info, "Request received")

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "a547276f-b02b-4e7d-b67e-c6deb07567da")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		expected := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")
		assert.Equal(t, expected, capturedRequestID)
	})

	t.Run("sets X-Request-ID response header", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, nil)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)
		logger := NewLogger(logConfig)

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := HTTPMiddlewareHandler(nextHandler, logger, DefaultLevels.Info, "Request received")

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "a547276f-b02b-4e7d-b67e-c6deb07567da")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, "a547276f-b02b-4e7d-b67e-c6deb07567da", rr.Header().Get("X-Request-ID"))
	})

	t.Run("creates new request ID when not provided", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, nil)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)
		logger := NewLogger(logConfig)

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := HTTPMiddlewareHandler(nextHandler, logger, DefaultLevels.Info, "Request received")

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// Should have set a new X-Request-ID header
		responseID := rr.Header().Get("X-Request-ID")
		assert.NotEmpty(t, responseID)
		assert.Len(t, responseID, 36) // UUID format
	})

	t.Run("logs request message", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, nil)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)
		logger := NewLogger(logConfig)

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := HTTPMiddlewareHandler(nextHandler, logger, DefaultLevels.Info, "Incoming request")

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		output := buf.String()
		assert.Contains(t, output, "Incoming request")
	})

	t.Run("works with nil logger", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := HTTPMiddlewareHandler(nextHandler, nil, DefaultLevels.Info, "Request received")

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		// Should not panic
		assert.NotPanics(t, func() {
			handler.ServeHTTP(rr, req)
		})
	})
}

func TestHTTPMiddlewareFunc(t *testing.T) {
	t.Run("returns middleware function", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		config := NewJSONWriterConfig(buf, nil)
		logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)
		logger := NewLogger(logConfig)

		middlewareFunc := HTTPMiddlewareFunc(logger, DefaultLevels.Info, "Request received")

		require.NotNil(t, middlewareFunc)

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := middlewareFunc(nextHandler)
		require.NotNil(t, handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "a547276f-b02b-4e7d-b67e-c6deb07567da")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, "a547276f-b02b-4e7d-b67e-c6deb07567da", rr.Header().Get("X-Request-ID"))
	})
}

func TestHTTPMiddlewareRespondPlaintextCtxLogsIfNotOK(t *testing.T) {
	t.Run("passes through OK response unchanged", func(t *testing.T) {
		wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"success"}`))
		})

		handler := HTTPMiddlewareRespondPlaintextCtxLogsIfNotOK(wrapped)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), `{"status":"success"}`)
	})

	t.Run("responds with logs on non-OK status", func(t *testing.T) {
		wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get logger from context and log something
			ctx := r.Context()
			buf := bytes.NewBuffer(nil)
			config := NewJSONWriterConfig(buf, nil)
			logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)
			logger := NewLogger(logConfig)

			logger.NewMessage(ctx, DefaultLevels.Error, "Something went wrong").Log()

			w.WriteHeader(http.StatusInternalServerError)
		})

		handler := HTTPMiddlewareRespondPlaintextCtxLogsIfNotOK(wrapped)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// Should set Content-Type to text/plain
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("appends text response body on error", func(t *testing.T) {
		wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Bad request error"))
		})

		handler := HTTPMiddlewareRespondPlaintextCtxLogsIfNotOK(wrapped)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Contains(t, rr.Body.String(), "Bad request error")
	})

	t.Run("appends JSON response body on error", func(t *testing.T) {
		wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"bad request"}`))
		})

		handler := HTTPMiddlewareRespondPlaintextCtxLogsIfNotOK(wrapped)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Contains(t, rr.Body.String(), `{"error":"bad request"}`)
	})

	t.Run("appends XML response body on error", func(t *testing.T) {
		wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`<error>bad request</error>`))
		})

		handler := HTTPMiddlewareRespondPlaintextCtxLogsIfNotOK(wrapped)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Contains(t, rr.Body.String(), `<error>bad request</error>`)
	})

	t.Run("copies headers from OK response", func(t *testing.T) {
		wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom-Header", "custom-value")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		})

		handler := HTTPMiddlewareRespondPlaintextCtxLogsIfNotOK(wrapped)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, "custom-value", rr.Header().Get("X-Custom-Header"))
	})

	t.Run("works with level filter", func(t *testing.T) {
		wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		filter := LevelFilterOutBelow(DefaultLevels.Warn)
		handler := HTTPMiddlewareRespondPlaintextCtxLogsIfNotOK(wrapped, filter)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		// Should not panic
		assert.NotPanics(t, func() {
			handler.ServeHTTP(rr, req)
		})
	})
}
