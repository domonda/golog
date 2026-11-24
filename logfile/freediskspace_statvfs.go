//go:build netbsd || solaris

package logfile

import "golang.org/x/sys/unix"

// systemFreeDiskSpace returns the bytes available to unprivileged users
// on the volume of path.
func systemFreeDiskSpace(path string) (int64, error) {
	var stat unix.Statvfs_t
	err := unix.Statvfs(path, &stat)
	if err != nil {
		return 0, err
	}
	// Bavail counts fragments of Frsize bytes.
	// Frsize and Bsize are uint32 on 32-bit NetBSD, so convert them
	blockSize := uint64(stat.Frsize)
	if blockSize == 0 {
		blockSize = uint64(stat.Bsize)
	}
	return clampToInt64(stat.Bavail * blockSize), nil
}
