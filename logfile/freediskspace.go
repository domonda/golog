package logfile

import (
	"math"
	"os"
)

// FreeDiskSpace returns the number of bytes that the calling process
// can still write to the volume (file system) that contains path,
// which can be a file or a directory that exists.
// Space reserved for the superuser is not included,
// so the result can be zero while the volume still has free blocks.
//
// It is supported on Windows and on Unix systems like Linux, macOS,
// the BSDs, Solaris, illumos and AIX. On other platforms it returns
// an error that wraps errors.ErrUnsupported.
// Errors are of type *os.PathError,
// use errors.Is with fs.ErrNotExist to detect a path that does not exist.
//
// Use it in a WriteCallback of a RotatingWriter to warn
// when the volume of the log file is running out of space,
// before writes fail because the disk is full.
// It makes one or two system calls, which is cheap enough
// to call after every Write, see the example for how to report
// low disk space only once instead of for every Write.
func FreeDiskSpace(path string) (int64, error) {
	free, err := freeDiskSpaceImpl(path)
	if err != nil {
		return 0, &os.PathError{Op: "free disk space", Path: path, Err: err}
	}
	return free, nil
}

// freeDiskSpaceImpl returns the bytes available to the calling process
// on the volume of path, implemented per platform by systemFreeDiskSpace.
// It is a variable so tests and examples can simulate a volume that fills up,
// see MockFreeDiskSpace in freediskspace_test.go.
var freeDiskSpaceImpl = systemFreeDiskSpace

// clampToInt64 converts a byte count reported by the operating system
// to int64, returning math.MaxInt64 for counts that don't fit.
func clampToInt64(n uint64) int64 {
	return int64(min(n, math.MaxInt64)) // #nosec G115 -- clamped to MaxInt64
}
