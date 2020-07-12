package golog

import "net/http"

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

// HTTPMiddlewareHandler returns a HTTP middleware handler that passes through a UUID requestID value.
// The requestID will be added as value to the http.Request before calling the next handler.
// If available the X-Request-ID or X-Correlation-ID HTTP request header will be used as requestID.
// It has to be a valid UUID in the format "994d5800-afca-401f-9c2f-d9e3e106e9ef".
// If the request has no requestID, then a random v4 UUID will be used.
// The requestID will also be set at the http.ResponseWriter as X-Request-ID header
// before calling the next handler, which has a chance to change it.
// If restrictHeaders are passed, then only those headers are logged if available,
// or pass HTTPNoHeaders to disable header logging.
// To disable logging of the request at all and just pass through
// the requestID pass LevelInvalid as log level.
// See also HTTPMiddlewareFunc.
func HTTPMiddlewareHandler(next http.Handler, logger *Logger, level Level, message string, restrictHeaders ...string) http.Handler {
	return http.HandlerFunc(
		func(response http.ResponseWriter, request *http.Request) {
			requestID := GetOrCreateRequestID(request)
			response.Header().Set("X-Request-ID", FormatUUID(requestID))

			requestWithID := AddValueToRequest(request, NewUUIDValue("requestID", requestID))

			logger.NewMessage(level, message).
				Request(requestWithID, restrictHeaders...).
				Log()

			next.ServeHTTP(response, requestWithID)
		},
	)
}

// HTTPMiddlewareFunc returns a HTTP middleware function that passes through a UUID requestID value.
// The requestID will be added as value to the http.Request before calling the next handler.
// If available the X-Request-ID or X-Correlation-ID HTTP request header will be used as requestID.
// It has to be a valid UUID in the format "994d5800-afca-401f-9c2f-d9e3e106e9ef".
// If the request has no requestID, then a random v4 UUID will be used.
// The requestID will also be set at the http.ResponseWriter as X-Request-ID header
// before calling the next handler, which has a chance to change it.
// If restrictHeaders are passed, then only those headers are logged if available,
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
