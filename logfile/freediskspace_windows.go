package logfile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// systemFreeDiskSpace returns the bytes available to the calling user
// on the volume of path, taking disk quotas into account.
func systemFreeDiskSpace(path string) (int64, error) {
	// GetDiskFreeSpaceEx needs a directory
	// and a UNC path needs a trailing backslash
	info, err := os.Stat(path)
	if err != nil {
		if pathErr, ok := errors.AsType[*os.PathError](err); ok {
			return 0, pathErr.Err
		}
		return 0, err
	}
	dir := path
	if !info.IsDir() {
		dir = filepath.Dir(path)
	}
	if !strings.HasSuffix(dir, `\`) && !strings.HasSuffix(dir, "/") {
		dir += `\`
	}
	dirUTF16, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return 0, err
	}
	var available uint64
	err = windows.GetDiskFreeSpaceEx(dirUTF16, &available, nil, nil)
	if err != nil {
		return 0, err
	}
	return clampToInt64(available), nil
}
