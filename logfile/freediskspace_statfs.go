//go:build aix || darwin || dragonfly || freebsd

package logfile

import "golang.org/x/sys/unix"

// systemFreeDiskSpace returns the bytes available to unprivileged users
// on the volume of path.
func systemFreeDiskSpace(path string) (int64, error) {
	var stat unix.Statfs_t
	err := unix.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}
	// Bavail is signed on FreeBSD and DragonFly, where it is negative
	// when the superuser has used up the reserved blocks
	return clampToInt64(uint64(max(stat.Bavail, 0)) * uint64(stat.Bsize)), nil // #nosec G115 -- clamped to zero
}
