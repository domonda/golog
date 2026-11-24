package logfile

import (
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockFreeDiskSpace makes FreeDiskSpace return the result of mock
// instead of asking the operating system, until the returned restore
// function is called. FreeDiskSpace still wraps errors of mock
// in an *os.PathError.
//
// It is defined in a _test.go file, so only the tests and examples
// of this package can use it, for example to simulate a volume
// that fills up with log files. It is not safe to call concurrently
// with FreeDiskSpace.
func MockFreeDiskSpace(mock func(path string) (int64, error)) (restore func()) {
	original := freeDiskSpaceImpl
	freeDiskSpaceImpl = mock
	return func() { freeDiskSpaceImpl = original }
}

func TestFreeDiskSpace(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "app.log")
	require.NoError(t, os.WriteFile(filePath, []byte("log\n"), 0o600))

	// The test can't know the free space, but a volume
	// that a temp file was just written to has some left
	dirFree, err := FreeDiskSpace(dir)
	require.NoError(t, err)
	assert.Positive(t, dirFree, "free space of the directory")

	fileFree, err := FreeDiskSpace(filePath)
	require.NoError(t, err)
	assert.Positive(t, fileFree, "free space of the volume of a file")

	// The file is on the volume of its directory, so both report about
	// the same free space, give or take what other processes wrote in between
	const tolerance = 1 << 30
	assert.InDelta(t, dirFree, fileFree, tolerance, "file and its directory are on the same volume")
}

func TestFreeDiskSpaceNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist", "app.log")
	free, err := FreeDiskSpace(path)
	assert.Zero(t, free)
	require.ErrorIs(t, err, fs.ErrNotExist)
	pathErr, ok := errors.AsType[*os.PathError](err)
	require.True(t, ok, "error is a *os.PathError")
	assert.Equal(t, "free disk space", pathErr.Op)
	assert.Equal(t, path, pathErr.Path)
	assert.Equal(t, "free disk space "+path+": "+pathErr.Err.Error(), err.Error())
}

func TestClampToInt64(t *testing.T) {
	assert.Equal(t, int64(0), clampToInt64(0))
	assert.Equal(t, int64(1<<40), clampToInt64(1<<40))
	assert.Equal(t, int64(math.MaxInt64), clampToInt64(math.MaxInt64))
	assert.Equal(t, int64(math.MaxInt64), clampToInt64(math.MaxInt64+1), "too large for int64")
	assert.Equal(t, int64(math.MaxInt64), clampToInt64(math.MaxUint64), "too large for int64")
}

func TestFreeDiskSpaceMock(t *testing.T) {
	restore := MockFreeDiskSpace(func(path string) (int64, error) {
		if path == "unsupported" {
			return 0, errors.ErrUnsupported
		}
		return 42, nil
	})
	defer restore()

	free, err := FreeDiskSpace("app.log")
	require.NoError(t, err)
	assert.Equal(t, int64(42), free)

	free, err = FreeDiskSpace("unsupported")
	assert.Zero(t, free)
	require.ErrorIs(t, err, errors.ErrUnsupported, "mock error is wrapped")
	pathErr, ok := errors.AsType[*os.PathError](err)
	require.True(t, ok, "mock error is wrapped in a *os.PathError")
	assert.Equal(t, "unsupported", pathErr.Path)

	restore()
	free, err = FreeDiskSpace(t.TempDir())
	require.NoError(t, err)
	assert.NotEqual(t, int64(42), free, "restore brings back the system implementation")
}
