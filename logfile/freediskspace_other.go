//go:build !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || windows)

package logfile

import (
	"errors"
	"fmt"
	"runtime"
)

// systemFreeDiskSpace returns an error that wraps errors.ErrUnsupported,
// because there is no implementation for this platform.
func systemFreeDiskSpace(path string) (int64, error) {
	return 0, fmt.Errorf("%w on %s", errors.ErrUnsupported, runtime.GOOS)
}
