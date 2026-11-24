package logfile

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// systemFreeDiskSpace returns the bytes available to unprivileged users
// on the volume of path.
func systemFreeDiskSpace(path string) (int64, error) {
	var stat unix.Statfs_t
	err := unix.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}
	// Bavail counts fragments of Frsize bytes, which only older kernels
	// don't report, and which equals Bsize for most file systems
	blockSize := stat.Frsize
	if blockSize <= 0 {
		blockSize = stat.Bsize
	}
	if blockSize <= 0 {
		return 0, fmt.Errorf("invalid block size %d", blockSize)
	}
	return clampToInt64(stat.Bavail * uint64(blockSize)), nil // #nosec G115 -- block size is positive
}
