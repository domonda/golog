package golog

import (
	"context"
	"net/http"
)

const HTTPNoHeaders = "HTTPNoHeaders"

// GetOrCreateRequestID gets a UUID from a http.Request or creates one.
// The X-Request-ID or X-Correlation-ID HTTP request headers will be
// parsed as UUID in the format "994d5800-afca-401f-9c2f-d9e3e106e9ef".
// If the request has no properly formatted ID,
// then a random v4 UUID will be returned.
func GetOrCreateRequestID(request *http.Request) [16]byte {
	xRequestID := request.Header.Get("X-Request-ID")
	if xRequestID == "" {
		xRequestID = request.Header.Get("X-Correlation-ID")
	}
	requestID, err := ParseUUID(xRequestID)
	if err != nil {
		return NewUUID()
	}
	return requestID
}

// GetRequestIDFromContext returns a UUID that was added
// to the context as UUID attribute with the key "requestID".
// If the context has no requestID attribute
// then false will be returned for ok.
func GetRequestIDFromContext(ctx context.Context) (requestID [16]byte, ok bool) {
	attrib, ok := AttribsFromContext(ctx).Get("requestID").(UUID)
	if !ok {
		return [16]byte{}, false
	}
	return attrib.Val, true
}

// GetRequestIDStringFromContext returns a UUID formatted as string
// that was added to the context as UUID attribute with the key "requestID".
// If the context has no requestID attribute
// then and empty string will be returned.
func GetRequestIDStringFromContext(ctx context.Context) string {
	requestID, ok := AttribsFromContext(ctx).Get("requestID").(UUID)
	if !ok {
		return ""
	}
	return FormatUUID(requestID.Val)
}

// GetOrCreateRequestIDFromContext returns a UUID that was added
// to the context as UUID attribute with the key "requestID"
// If the context has no requestID attribute
// then a new random v4 UUID will be returned.
func GetOrCreateRequestIDFromContext(ctx context.Context) [16]byte {
	requestID, ok := AttribsFromContext(ctx).Get("requestID").(UUID)
	if !ok {
		return NewUUID()
	}
	return requestID.Val
}

// ContextWithRequestID adds the passed requestID as UUID
// attribute with the key "requestID" to the context.
func ContextWithRequestID(ctx context.Context, requestID [16]byte) context.Context {
	return ContextWithAttribs(ctx, UUID{Key: "requestID", Val: requestID})
}

// HTTPMiddlewareHandler returns a HTTP middleware handler that passes through a UUID requestID.
// The requestID will be added as UUID Attrib to the http.Request before calling the next handler.
// If available the X-Request-ID or X-Correlation-ID HTTP request header will be used as requestID.
// It has to be a valid UUID in the format "994d5800-afca-401f-9c2f-d9e3e106e9ef".
// If the request has no requestID, then a random v4 UUID will be used.
// The requestID will also be set at the http.ResponseWriter as X-Request-ID header
// before calling the next handler, which has a chance to change it.
// If restrictHeaders are passed then only those headers are logged if available,
// or pass HTTPNoHeaders to disable header logging.
// To disable logging of the request at all and just pass through
// the requestID pass LevelInvalid as log level.
// See also HTTPMiddlewareFunc.
func HTTPMiddlewareHandler(next http.Handler, logger *Logger, level Level, message string, restrictHeaders ...string) http.Handler {
	return http.HandlerFunc(
		func(response http.ResponseWriter, request *http.Request) {
			requestID := GetOrCreateRequestID(request)
			response.Header().Set("X-Request-ID", FormatUUID(requestID))

			requestWithID := RequestWithAttribs(request, UUID{Key: "requestID", Val: requestID})

			logger.NewMessage(level, message).
				Request(requestWithID, restrictHeaders...).
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
// If restrictHeaders are passed then only those headers are logged if available,
// or pass HTTPNoHeaders to disable header logging.
// To disable logging of the request at all and just pass through
// the requestID pass LevelInvalid as log level.
// Compatible with github.com/gorilla/mux.MiddlewareFunc.
// See also HTTPMiddlewareHandler.
func HTTPMiddlewareFunc(logger *Logger, level Level, message string, restrictHeaders ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return HTTPMiddlewareHandler(next, logger, level, message, restrictHeaders...)
	}
}
