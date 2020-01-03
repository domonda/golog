package golog

var (
	// HTTPRequestLevel is used for logging HTTP requests with Logger.WithRequest or Logger.WithRequestContext
	HTTPRequestLevel = &DefaultLevels.Info

	// HTTPRequestMessage is used for logging HTTP requests with Logger.WithRequest or Logger.WithRequestContext
	HTTPRequestMessage = "HTTP request"
)
