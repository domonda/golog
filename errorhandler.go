package golog

import (
	"fmt"
	"os"
)

// ErrorHandler will be called when an
// error occured while writing the logs.
// The default handler prints to stderr.
var ErrorHandler = func(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
}
