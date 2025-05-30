package golog

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
)

const HTTPNoHeaders = "HTTPNoHeaders"

// GetOrCreateRequestUUID gets a UUID from a http.Request or creates one.
// The X-Request-ID or X-Correlation-ID HTTP request headers will be
// parsed as UUID in the format "994d5800-afca-401f-9c2f-d9e3e106e9ef".
// If the request has no properly formatted ID,
// then a random v4 UUID will be returned.
func GetOrCreateRequestUUID(request *http.Request) [16]byte {
	xRequestID := request.Header.Get("X-Request-ID")
	if xRequestID == "" {
		xRequestID = request.Header.Get("X-Correlation-ID")
	}
	requestID, err := ParseUUID(xRequestID)
	if err != nil {
		return UUIDv4()
	}
	return requestID
}

// GetOrCreateRequestID gets a string from a http.Request or creates
// one as formatted random UUID.
// The X-Request-ID or X-Correlation-ID HTTP request header values
// will be returned if available,
// else a random v4 UUID will be returned formatted as string.
func GetOrCreateRequestID(request *http.Request) string {
	requestID := request.Header.Get("X-Request-ID")
	if requestID != "" {
		return requestID
	}
	requestID = request.Header.Get("X-Correlation-ID")
	if requestID != "" {
		return requestID
	}
	return FormatUUID(UUIDv4())
}

// GetRequestUUIDFromContext returns a UUID that was added
// to the context as UUID attribute with the key "requestID".
// If the context has no requestID attribute
// then false will be returned for ok.
func GetRequestUUIDFromContext(ctx context.Context) (requestID [16]byte, ok bool) {
	attrib, ok := AttribsFromContext(ctx).Get("requestID").(*UUID)
	if !ok {
		return [16]byte{}, false
	}
	return attrib.val, true
}

// GetRequestIDFromContext returns a string
// that was added to the context as attribute with the key "requestID".
// If the context has no requestID attribute
// then and empty string will be returned.
func GetRequestIDFromContext(ctx context.Context) string {
	requestID := AttribsFromContext(ctx).Get("requestID")
	if requestID == nil {
		return ""
	}
	return requestID.ValueString()
}

// GetOrCreateRequestUUIDFromContext returns a UUID that was added
// to the context as UUID attribute with the key "requestID"
// If the context has no requestID attribute
// then a new random v4 UUID will be returned.
func GetOrCreateRequestUUIDFromContext(ctx context.Context) [16]byte {
	requestID, ok := AttribsFromContext(ctx).Get("requestID").(*UUID)
	if !ok {
		return UUIDv4()
	}
	return requestID.val
}

// ContextWithRequestUUID adds the passed requestID as UUID
// attribute with the key "requestID" to the context.
func ContextWithRequestUUID(ctx context.Context, requestID [16]byte) context.Context {
	return ContextWithAttribs(ctx, NewUUID("requestID", requestID))
}

// ContextWithRequestID adds the passed requestID as string
// attribute with the key "requestID" to the context.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return ContextWithAttribs(ctx, NewString("requestID", requestID))
}

// HTTPMiddlewareHandler returns a HTTP middleware handler that passes through a UUID requestID.
// The requestID will be added as UUID Attrib to the http.Request before calling the next handler.
// If available the X-Request-ID or X-Correlation-ID HTTP request header will be used as requestID.
// It has to be a valid UUID in the format "994d5800-afca-401f-9c2f-d9e3e106e9ef".
// If the request has no requestID, then a random v4 UUID will be used.
// The requestID will also be set at the http.ResponseWriter as X-Request-ID header
// before calling the next handler, which has a chance to change it.
// If onlyHeaders are passed then only those headers are logged if available,
// or pass HTTPNoHeaders to disable header logging.
// To disable logging of the request at all and just pass through
// the requestID pass LevelInvalid as log level.
// See also HTTPMiddlewareFunc.
func HTTPMiddlewareHandler(next http.Handler, logger *Logger, level Level, message string, onlyHeaders ...string) http.Handler {
	return http.HandlerFunc(
		func(response http.ResponseWriter, request *http.Request) {
			requestID := GetOrCreateRequestUUID(request)
			response.Header().Set("X-Request-ID", FormatUUID(requestID))

			requestWithID := RequestWithAttribs(request, NewUUID("requestID", requestID))

			logger.NewMessage(request.Context(), level, message).
				Request(requestWithID, onlyHeaders...).
				Log()

			next.ServeHTTP(response, requestWithID)
		},
	)
}

// HTTPMiddlewareFunc returns a HTTP middleware function that passes through a UUID requestID.
// The requestID will be added as UUID Attrib to the http.Request before calling the next handler.
// If available the X-Request-ID or X-Correlation-ID HTTP request header will be used as requestID.
// It has to be a valid UUID in the format "994d5800-afca-401f-9c2f-d9e3e106e9ef".
// If the request has no requestID, then a random v4 UUID will be used.
// The requestID will also be set at the http.ResponseWriter as X-Request-ID header
// before calling the next handler, which has a chance to change it.
// If onlyHeaders are passed then only those headers are logged if available,
// or pass HTTPNoHeaders to disable header logging.
// To disable logging of the request at all and just pass through
// the requestID pass LevelInvalid as log level.
// Compatible with github.com/gorilla/mux.MiddlewareFunc.
// See also HTTPMiddlewareHandler.
func HTTPMiddlewareFunc(logger *Logger, level Level, message string, onlyHeaders ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return HTTPMiddlewareHandler(next, logger, level, message, onlyHeaders...)
	}
}

// HTTPMiddlewareRespondPlaintextCtxLogsIfNotOK adds a TextWriterConfig to the request context
// that collects all log messages that are created with the passed through context
// and writes an alternative plaintext response with the collected logs
// if the response status code from the wrapped handler is not 200 OK.
// In case of a status code 200 OK response from the wrapped handler
// the original response will be passed through.
func HTTPMiddlewareRespondPlaintextCtxLogsIfNotOK(wrapped http.Handler, filter ...LevelFilter) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		var (
			logBuffer  = bytes.NewBuffer(nil)
			logContext = ContextWithAdditionalWriterConfigs(
				request.Context(),
				NewTextWriterConfig(logBuffer, nil, nil, filter...),
			)
			responseRecorder = httptest.NewRecorder()
		)

		wrapped.ServeHTTP(responseRecorder, request.WithContext(logContext))

		if responseRecorder.Code != http.StatusOK {
			// In case of an error respond with the recorded logs
			response.Header().Set("Content-Type", "text/plain; charset=utf-8")
			response.Header().Set("X-Content-Type-Options", "nosniff")
			response.Write(logBuffer.Bytes())

			// Also respond with the recorded response body if it is text
			ct := responseRecorder.Header().Get("Content-Type")
			if strings.HasPrefix(ct, "text/") || strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "application/xml") {
				response.Write([]byte("\n\n"))
				response.Write(responseRecorder.Body.Bytes())
				return
			}
		}

		// Response was OK, copy recorded response
		// to actual to response
		for k, vals := range responseRecorder.Header() {
			for _, v := range vals {
				response.Header().Add(k, v)
			}
		}
		response.Write(responseRecorder.Body.Bytes())
	}
}
