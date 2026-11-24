package logfile

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/golog"
	"github.com/domonda/golog/log"
)

// listDir returns the paths of the entries in dir whose names match pattern.
// Unlike filepath.Glob it only matches pattern against the entry names,
// so glob metacharacters in dir (e.g. from TMPDIR) are not interpreted,
// and it returns directory read errors instead of ignoring them.
func listDir(dir, pattern string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		match, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			return nil, err
		}
		if match {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}
	return paths, nil
}

func TestRotatingWriter_BasicWrite(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 0)
	require.NoError(t, err)
	defer writer.Close()

	// Write some data
	data := []byte("Hello, World!\n")
	n, err := writer.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)

	// Verify file exists and contains data
	err = writer.Sync()
	require.NoError(t, err)

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, string(data), string(content))
}

func TestRotatingWriter_NoRotation(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	// rotateSize = 0 means no rotation
	writer, err := NewRotatingWriter(filePath, "", 0644, 0)
	require.NoError(t, err)
	defer writer.Close()

	// Write data that would trigger rotation if enabled
	data := []byte(strings.Repeat("This is a test log line.\n", 100))
	n, err := writer.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)

	err = writer.Sync()
	require.NoError(t, err)

	// Verify only one file exists
	files, err := listDir(dir, "test.log*")
	require.NoError(t, err)
	assert.Equal(t, 1, len(files), "should only have one file with no rotation")
}

func TestRotatingWriter_SingleRotation(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	rotateSize := int64(100)
	writer, err := NewRotatingWriter(filePath, "", 0644, rotateSize)
	require.NoError(t, err)
	defer writer.Close()

	// First write - should not rotate
	data1 := []byte(strings.Repeat("A", 50))
	n, err := writer.Write(data1)
	require.NoError(t, err)
	assert.Equal(t, len(data1), n)

	// Second write - should trigger rotation
	data2 := []byte(strings.Repeat("B", 60))
	n, err = writer.Write(data2)
	require.NoError(t, err)
	assert.Equal(t, len(data2), n)

	err = writer.Sync()
	require.NoError(t, err)

	// Should have 2 files: original and rotated
	files, err := listDir(dir, "test.log*")
	require.NoError(t, err)
	assert.Equal(t, 2, len(files), "should have original and one rotated file")

	// Verify the current file contains only the second write
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, string(data2), string(content))
}

func TestRotatingWriter_MultipleRotations(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	rotateSize := int64(50)
	writer, err := NewRotatingWriter(filePath, "", 0644, rotateSize)
	require.NoError(t, err)
	defer writer.Close()

	// Write multiple times to trigger multiple rotations
	// Each write is 60 bytes, rotation happens at 50 bytes
	// First write: 60 bytes, no rotation (file has 60 bytes)
	// Second write: would be 120 bytes, rotation happens first, then write (file has 60 bytes, 1 rotated file)
	// Third write: would be 120 bytes, rotation happens first, then write (file has 60 bytes, 2 rotated files)
	// etc.
	numWrites := 5
	for i := range numWrites {
		data := []byte(strings.Repeat("X", 60))
		_, err := writer.Write(data)
		require.NoError(t, err, "write %d failed", i)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	err = writer.Sync()
	require.NoError(t, err)

	// Should have: 1 current file + (numWrites-1) rotated files = numWrites total files
	// But actually: first write doesn't rotate, subsequent writes each rotate before writing
	// So we get: numWrites+1 files (current + numWrites rotated)
	// Actually, each write triggers rotation if size >= rotateSize
	// So: write 1 (60 bytes), write 2 (rotate, 60 bytes), write 3 (rotate, 60 bytes), etc.
	files, err := listDir(dir, "test.log*")
	require.NoError(t, err)
	expectedFiles := numWrites + 1 // Current file + one rotated file per write after first
	assert.Equal(t, expectedFiles, len(files), "should have %d files after %d writes", expectedFiles, numWrites)
}

func TestRotatingWriter_CustomTimeFormat(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	customFormat := "2006-01-02"
	rotateSize := int64(50)
	writer, err := NewRotatingWriter(filePath, customFormat, 0644, rotateSize)
	require.NoError(t, err)
	defer writer.Close()

	assert.Equal(t, customFormat, writer.TimeFormat())

	// Trigger rotation
	data := []byte(strings.Repeat("Y", 60))
	_, err = writer.Write(data)
	require.NoError(t, err)

	err = writer.Sync()
	require.NoError(t, err)

	// The single write rotates once, before writing
	files, err := listDir(dir, "test.log.*")
	require.NoError(t, err)
	require.Len(t, files, 1, "should have exactly one rotated file")

	// The suffix must parse with the custom format, like test.log.2024-01-15.
	// A suffix in the default format (2006-01-02_15:04:05) fails to parse
	// with "extra text", and parsing does not depend on the current time.
	suffix := strings.TrimPrefix(filepath.Base(files[0]), "test.log.")
	_, err = time.Parse(customFormat, suffix)
	require.NoError(t, err, "rotated file suffix %q should use the custom format %q", suffix, customFormat)
}

func TestRotatingWriter_ExistingFile(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "existing.log")

	// Create an existing file with content
	existingContent := []byte("Existing log content\n")
	err := os.WriteFile(filePath, existingContent, 0644)
	require.NoError(t, err)

	// Open with RotatingWriter
	rotateSize := int64(100)
	writer, err := NewRotatingWriter(filePath, "", 0644, rotateSize)
	require.NoError(t, err)
	defer writer.Close()

	// Write new content
	newContent := []byte("New log content\n")
	_, err = writer.Write(newContent)
	require.NoError(t, err)

	err = writer.Sync()
	require.NoError(t, err)

	// Verify both contents are present (append mode)
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, string(existingContent)+string(newContent), string(content))
}

func TestRotatingWriter_Sync(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 0)
	require.NoError(t, err)
	defer writer.Close()

	data := []byte("Test data\n")
	_, err = writer.Write(data)
	require.NoError(t, err)

	// Sync should flush to disk
	err = writer.Sync()
	require.NoError(t, err)

	// Verify data is on disk
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, string(data), string(content))
}

func TestRotatingWriter_Getters(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	timeFormat := "2006-01-02_15:04"
	rotateSize := int64(1024 * 1024)

	writer, err := NewRotatingWriter(filePath, timeFormat, 0644, rotateSize)
	require.NoError(t, err)
	defer writer.Close()

	assert.Equal(t, filePath, writer.FilePath())
	assert.Equal(t, timeFormat, writer.TimeFormat())
	assert.Equal(t, rotateSize, writer.RotateSize())
}

func TestRotatingWriter_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "concurrent.log")
	rotateSize := int64(200)
	writer, err := NewRotatingWriter(filePath, "", 0644, rotateSize)
	require.NoError(t, err)
	defer writer.Close()

	numGoroutines := 10
	numWritesPerGoroutine := 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for j := range numWritesPerGoroutine {
				data := []byte(strings.Repeat("X", 30))
				_, err := writer.Write(data)
				assert.NoError(t, err, "goroutine %d, write %d failed", id, j)
			}
		}(i)
	}

	wg.Wait()

	err = writer.Sync()
	require.NoError(t, err)

	// Verify files were created (should have rotated multiple times)
	files, err := listDir(dir, "concurrent.log*")
	require.NoError(t, err)
	assert.Greater(t, len(files), 1, "should have rotated at least once with concurrent writes")
}

// TestRotatingWriter_WithGolog tests integration with golog using concurrent writes
// This is the original test, now improved with better assertions
func TestRotatingWriter_WithGolog(t *testing.T) {
	dir := t.TempDir()

	const jsonRotateSize = 350 // fit two 165 byte lines, but not three
	jsonWriter, err := NewRotatingWriter(
		filepath.Join(dir, "json.log"), "", 0644, jsonRotateSize)
	require.NoError(t, err)
	defer jsonWriter.Close()

	const textRotateSize = 300 // fit two 130 byte lines but not three
	textWriter, err := NewRotatingWriter(
		filepath.Join(dir, "text.log"), "", 0644, textRotateSize)
	require.NoError(t, err)
	defer textWriter.Close()

	// Create a local logger config instead of using global log.Config
	logConfig := golog.NewConfig(
		log.Levels,
		golog.AllLevelsActive,
		golog.NewJSONWriterConfig(jsonWriter, &log.Format),
		golog.NewTextWriterConfig(textWriter, &log.Format, golog.NoColorizer),
	)

	logger := golog.NewLogger(logConfig)

	numThreads := 8
	numThreadMessages := 66

	var wg sync.WaitGroup
	wg.Add(numThreads)

	for i := range numThreads {
		go func(thread int) {
			for threadMsg := range numThreadMessages {
				logger.Info("Thread log").
					Int("thread", thread).
					Int("threadMsg", threadMsg).
					Str("filler", "Filler to over 100 bytes per log line in JSON and text").
					Log()
				time.Sleep(time.Millisecond)
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	// Flush logs to ensure all data is written
	logger.Flush()

	numMessagesPerFile := 2
	numFilesExpected := numThreads * numThreadMessages / numMessagesPerFile

	// Verify JSON log rotation
	jsonFiles, err := listDir(dir, "json.log*")
	require.NoError(t, err)
	require.Equal(t, numFilesExpected, len(jsonFiles), "expected %d files for json.log*, got %d", numFilesExpected, len(jsonFiles))

	// Verify text log rotation
	textFiles, err := listDir(dir, "text.log*")
	require.NoError(t, err)
	require.Equal(t, numFilesExpected, len(textFiles), "expected %d files for text.log*, got %d", numFilesExpected, len(textFiles))

	// Verify that all log messages were written
	totalMessages := numThreads * numThreadMessages
	// Count messages in JSON files
	jsonMessageCount := 0
	for _, file := range jsonFiles {
		content, err := os.ReadFile(file)
		require.NoError(t, err)
		// Each line is a JSON log message
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		jsonMessageCount += len(lines)
	}
	assert.Equal(t, totalMessages, jsonMessageCount, "should have all messages in JSON files")
}

// TestRotatingWriter_RecoveryAfterFileDeleted verifies that
// a failed rotation (e.g., file deleted externally) does not leave
// rw.file pointing to a closed file descriptor. The writer recovers
// by opening the file path again, writes the message of the failed
// rotation, and remains functional for subsequent writes.
func TestRotatingWriter_RecoveryAfterFileDeleted(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 100)
	require.NoError(t, err)
	defer writer.Close()

	// Write some data below the rotation threshold
	_, err = writer.Write([]byte(strings.Repeat("A", 50)))
	require.NoError(t, err)

	// Delete the file externally while the writer still has the fd open.
	// On Unix, the fd remains valid but the directory entry is removed.
	err = os.Remove(filePath)
	require.NoError(t, err)

	// Write enough to trigger rotation.
	// rotate() will: close fd (ok), rename path (fails: ENOENT),
	// then writeFile opens the path again (ok: O_CREATE) and writes the message.
	// Write returns the rename error, but the message is not lost.
	failed := []byte(strings.Repeat("B", 60))
	n, err := writer.Write(failed)
	var rotateErr *RotateError
	require.ErrorAs(t, err, &rotateErr, "write should return rename error from failed rotation")
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.Equal(t, len(failed), n, "message of the failed rotation is written")

	// The writer must remain functional after the failed rotation
	data := []byte("recovered\n")
	n, err = writer.Write(data)
	require.NoError(t, err, "writer should be functional after recovery")
	assert.Equal(t, len(data), n)

	require.NoError(t, writer.Sync())

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, string(failed)+string(data), string(content))
}

func TestRotatingWriter_SameTimestampSuffix(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	// Day-precision format guarantees all rotations within this test
	// produce the same timestamp string, exercising the numeric suffix logic.
	writer, err := NewRotatingWriter(filePath, "2006-01-02", 0644, 10)
	require.NoError(t, err)
	defer writer.Close()

	numRotations := 5
	for i := range numRotations {
		_, err := writer.Write([]byte(strings.Repeat("X", 20)))
		require.NoError(t, err, "write %d failed", i)
	}

	require.NoError(t, writer.Sync())

	files, err := listDir(dir, "test.log*")
	require.NoError(t, err)
	// 1 current file + numRotations rotated files
	// (each write of 20 bytes >= rotateSize of 10, so every write rotates)
	require.Equal(t, numRotations+1, len(files))

	// Collect rotated file names (everything except "test.log")
	today := time.Now().Format("2006-01-02")
	expectedBase := "test.log." + today

	rotatedNames := make(map[string]bool)
	for _, f := range files {
		if filepath.Base(f) != "test.log" {
			rotatedNames[filepath.Base(f)] = true
		}
	}
	assert.Equal(t, numRotations, len(rotatedNames))

	// First rotation gets the base name, subsequent ones get .1, .2, ...
	assert.True(t, rotatedNames[expectedBase], "should have base rotated file %s", expectedBase)
	for i := 1; i < numRotations; i++ {
		suffixed := expectedBase + "." + strconv.Itoa(i)
		assert.True(t, rotatedNames[suffixed], "should have suffixed rotated file %s", suffixed)
	}
}

func TestRotatingWriter_ExactBoundary(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 100)
	require.NoError(t, err)
	defer writer.Close()

	// A single write of exactly rotateSize bytes should trigger rotation
	data := []byte(strings.Repeat("A", 100))
	n, err := writer.Write(data)
	require.NoError(t, err)
	assert.Equal(t, 100, n)

	require.NoError(t, writer.Sync())

	files, err := listDir(dir, "test.log*")
	require.NoError(t, err)
	assert.Equal(t, 2, len(files), "should have current + 1 rotated file at exact boundary")

	// Current file should contain the data (written after rotation)
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, string(data), string(content))
}

func TestRotatingWriter_WriteLargerThanRotateSize(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 50)
	require.NoError(t, err)
	defer writer.Close()

	// Single write much larger than rotateSize.
	// Rotation happens, then the entire message is written to the new file.
	data := []byte(strings.Repeat("X", 200))
	n, err := writer.Write(data)
	require.NoError(t, err)
	assert.Equal(t, 200, n)

	require.NoError(t, writer.Sync())

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, string(data), string(content))

	// Rotated file should be empty (the original file was freshly created)
	rotated, err := listDir(dir, "test.log.*")
	require.NoError(t, err)
	require.Equal(t, 1, len(rotated))

	rotatedContent, err := os.ReadFile(rotated[0])
	require.NoError(t, err)
	assert.Empty(t, rotatedContent, "rotated file should be empty since original was fresh")
}

func TestRotatingWriter_RotatedFileContent(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 100)
	require.NoError(t, err)
	defer writer.Close()

	// Write data that stays below the rotation threshold
	data1 := []byte("first message\n")
	_, err = writer.Write(data1)
	require.NoError(t, err)

	// This write pushes the accumulated size past rotateSize,
	// so data1 ends up in the rotated file and data2 in the current file.
	data2 := []byte(strings.Repeat("B", 90) + "\n")
	_, err = writer.Write(data2)
	require.NoError(t, err)

	require.NoError(t, writer.Sync())

	// Verify rotated file has data1
	rotated, err := listDir(dir, "test.log.*")
	require.NoError(t, err)
	require.Equal(t, 1, len(rotated))

	rotatedContent, err := os.ReadFile(rotated[0])
	require.NoError(t, err)
	assert.Equal(t, string(data1), string(rotatedContent), "rotated file should contain first message")

	// Verify current file has data2
	currentContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, string(data2), string(currentContent), "current file should contain second message")
}

// writeCallbackCall records the arguments of one WriteCallback call
type writeCallbackCall struct {
	filePath string
	data     string
	n        int
	err      error
}

// recordWriteCallbacks sets a WriteCallback on writer
// that records the arguments of every call.
func recordWriteCallbacks(writer *RotatingWriter) *[]writeCallbackCall {
	var calls []writeCallbackCall
	writer.SetWriteCallback(func(filePath string, data []byte, n int, err error) {
		calls = append(calls, writeCallbackCall{filePath: filePath, data: string(data), n: n, err: err})
	})
	return &calls
}

func TestRotatingWriter_WriteCallback(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 100)
	require.NoError(t, err)
	defer writer.Close()

	calls := recordWriteCallbacks(writer)

	data1 := []byte("first message\n")
	_, err = writer.Write(data1)
	require.NoError(t, err)

	// Rotates the file before writing
	data2 := []byte(strings.Repeat("B", 90) + "\n")
	_, err = writer.Write(data2)
	require.NoError(t, err)

	assert.Equal(t, []writeCallbackCall{
		{filePath: filePath, data: string(data1), n: len(data1)},
		{filePath: filePath, data: string(data2), n: len(data2)},
	}, *calls)

	// No calls after removing the callback
	writer.SetWriteCallback(nil)
	_, err = writer.Write([]byte("not recorded\n"))
	require.NoError(t, err)
	assert.Len(t, *calls, 2)
}

func TestNewRotatingWriterWithCallback(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	var calls []writeCallbackCall
	writer, err := NewRotatingWriterWithCallback(filePath, "", 0644, 100,
		func(filePath string, data []byte, n int, err error) {
			calls = append(calls, writeCallbackCall{filePath: filePath, data: string(data), n: n, err: err})
		},
	)
	require.NoError(t, err)
	defer writer.Close()

	assert.Equal(t, filePath, writer.FilePath())
	assert.Equal(t, RotatingWriterDefaultTimeFormat, writer.TimeFormat())
	assert.Equal(t, int64(100), writer.RotateSize())

	// The callback is set before the first write
	data := []byte("first message\n")
	_, err = writer.Write(data)
	require.NoError(t, err)
	assert.Equal(t, []writeCallbackCall{{filePath: filePath, data: string(data), n: len(data)}}, calls)

	// A nil callback works like NewRotatingWriter
	nilWriter, err := NewRotatingWriterWithCallback(filepath.Join(dir, "nil.log"), "", 0644, 0, nil)
	require.NoError(t, err)
	defer nilWriter.Close()
	_, err = nilWriter.Write(data)
	require.NoError(t, err)
}

func TestRotatingWriter_WriteCallbackRotationError(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 100)
	require.NoError(t, err)
	defer writer.Close()

	_, err = writer.Write([]byte(strings.Repeat("A", 50)))
	require.NoError(t, err)

	calls := recordWriteCallbacks(writer)

	// Renaming the deleted file fails when the next write rotates it
	err = os.Remove(filePath)
	require.NoError(t, err)

	data := []byte(strings.Repeat("B", 60))
	n, err := writer.Write(data)
	var rotateErr *RotateError
	require.ErrorAs(t, err, &rotateErr)
	assert.Equal(t, len(data), n, "data is written despite the failed rotation")

	require.Len(t, *calls, 1)
	assert.Equal(t, filePath, (*calls)[0].filePath)
	assert.Equal(t, string(data), (*calls)[0].data, "callback gets the data passed to Write")
	assert.Equal(t, len(data), (*calls)[0].n, "callback gets n=len(data) because the data was written")
	assert.Equal(t, err, (*calls)[0].err, "callback gets the error returned by Write")
	assert.ErrorAs(t, (*calls)[0].err, &rotateErr, "callback can detect the rotation error")
}

func TestRotatingWriter_WriteCallbackWriteError(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 0)
	require.NoError(t, err)

	calls := recordWriteCallbacks(writer)

	// Writing to the closed file fails
	require.NoError(t, writer.Close())

	data := []byte("after close\n")
	n, err := writer.Write(data)
	require.Error(t, err)

	require.Len(t, *calls, 1)
	assert.Equal(t, filePath, (*calls)[0].filePath)
	assert.Equal(t, string(data), (*calls)[0].data)
	assert.Equal(t, n, (*calls)[0].n, "callback gets the byte count returned by Write")
	assert.Equal(t, err, (*calls)[0].err)
}

func TestRotatingWriter_WriteCallbackCanUseWriter(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 0)
	require.NoError(t, err)
	defer writer.Close()

	// Sync locks the writer's mutex, so it would deadlock
	// if the callback were called while Write holds the mutex
	var syncErr error
	writer.SetWriteCallback(func(string, []byte, int, error) {
		syncErr = writer.Sync()
	})

	data := []byte("synced in callback\n")
	_, err = writer.Write(data)
	require.NoError(t, err)
	require.NoError(t, syncErr)

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, string(data), string(content))
}

func TestRotatingWriter_WriteCallbackConcurrent(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 1000)
	require.NoError(t, err)
	defer writer.Close()

	var (
		mtx      sync.Mutex
		numCalls int
		numBytes int
	)
	writer.SetWriteCallback(func(_ string, data []byte, n int, err error) {
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		mtx.Lock()
		numCalls++
		numBytes += n
		mtx.Unlock()
	})

	const numGoroutines, numWrites = 10, 100
	data := []byte("concurrent log line\n")
	var wg sync.WaitGroup
	for range numGoroutines {
		wg.Go(func() {
			for range numWrites {
				_, err := writer.Write(data)
				assert.NoError(t, err)
			}
		})
	}
	wg.Wait()

	assert.Equal(t, numGoroutines*numWrites, numCalls)
	assert.Equal(t, numGoroutines*numWrites*len(data), numBytes)
}

func TestRotatingWriter_WriteCallbackPanic(t *testing.T) {
	var handledErrs []error
	prevHandler := golog.ErrorHandler()
	golog.SetErrorHandler(func(err error) { handledErrs = append(handledErrs, err) })
	t.Cleanup(func() { golog.SetErrorHandler(prevHandler) })

	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 0)
	require.NoError(t, err)
	defer writer.Close()

	writer.SetWriteCallback(func(string, []byte, int, error) {
		panic("callback panic")
	})

	// The panic is recovered and does not change the result of Write
	data1 := []byte("first message\n")
	n, err := writer.Write(data1)
	require.NoError(t, err)
	assert.Equal(t, len(data1), n)

	require.Len(t, handledErrs, 1)
	assert.ErrorContains(t, handledErrs[0], "callback panic")
	assert.ErrorContains(t, handledErrs[0], filePath)

	// The writer is still usable after the panic
	data2 := []byte("second message\n")
	_, err = writer.Write(data2)
	require.NoError(t, err)
	assert.Len(t, handledErrs, 2)

	// A disabled error handler drops the recovered panic
	golog.SetErrorHandler(nil)
	_, err = writer.Write([]byte("third message\n"))
	require.NoError(t, err)
	assert.Len(t, handledErrs, 2)

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "first message\nsecond message\nthird message\n", string(content))
}

// TestRotatingWriter_WriteAllocs checks that Write does not allocate,
// neither without a callback nor for calling a callback.
func TestRotatingWriter_WriteAllocs(t *testing.T) {
	dir := t.TempDir()

	writer, err := NewRotatingWriter(filepath.Join(dir, "test.log"), "", 0644, 0)
	require.NoError(t, err)
	defer writer.Close()

	msg := []byte("message\n")
	allocs := testing.AllocsPerRun(100, func() {
		_, _ = writer.Write(msg)
	})
	assert.Zero(t, allocs, "Write without callback")

	var calls int
	writer.SetWriteCallback(func(string, []byte, int, error) { calls++ })
	allocs = testing.AllocsPerRun(100, func() {
		_, _ = writer.Write(msg)
	})
	assert.Zero(t, allocs, "Write with callback")
	assert.Equal(t, 101, calls, "AllocsPerRun calls the function once more as warm-up")
}

// failRename makes renaming the log file fail until the test ends.
func failRename(t *testing.T) {
	t.Helper()
	renameFile = func(oldpath, newpath string) error {
		return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: os.ErrPermission}
	}
	t.Cleanup(func() { renameFile = os.Rename })
}

// TestRotatingWriter_RenameFailureKeepsWriting verifies that a log file
// that can't be renamed keeps all written data instead of dropping
// every write after the file reached the rotation size.
func TestRotatingWriter_RenameFailureKeepsWriting(t *testing.T) {
	dir := t.TempDir()
	failRename(t)

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 100)
	require.NoError(t, err)
	defer writer.Close()

	calls := recordWriteCallbacks(writer)

	var want strings.Builder
	for i := range 10 {
		data := []byte(strings.Repeat(strconv.Itoa(i), 59) + "\n")
		want.Write(data)
		n, err := writer.Write(data)
		assert.Equal(t, len(data), n, "write %d", i)
		switch i {
		case 0:
			// 60 bytes, below the rotation size
			require.NoError(t, err, "write %d", i)
		case 1:
			// 120 bytes, the failing rotation is reported once
			var rotateErr *RotateError
			require.ErrorAs(t, err, &rotateErr, "write %d", i)
			assert.Equal(t, filePath, rotateErr.FilePath)
			assert.ErrorIs(t, err, os.ErrPermission)
			assert.Equal(t, err, rotateErr, "RotateError is not joined when the write succeeded")
		default:
			// The rotation is not retried within the retry interval
			require.NoError(t, err, "write %d", i)
		}
	}
	require.Len(t, *calls, 10)
	var rotateErr *RotateError
	assert.ErrorAs(t, (*calls)[1].err, &rotateErr, "callback gets the rotation error")
	assert.Equal(t, 60, (*calls)[1].n, "callback of the failed rotation gets n=len(data)")

	require.NoError(t, writer.Sync())
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, want.String(), string(content), "no write is lost")

	files, err := listDir(dir, "test.log*")
	require.NoError(t, err)
	assert.Len(t, files, 1, "no rotated file was created")
}

// TestRotatingWriter_RotationRetryAfterFailure verifies that a failed
// rotation is retried after the retry interval and succeeds
// once renaming works again.
func TestRotatingWriter_RotationRetryAfterFailure(t *testing.T) {
	dir := t.TempDir()
	failRename(t)

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 100)
	require.NoError(t, err)
	defer writer.Close()

	first := []byte(strings.Repeat("A", 60))
	_, err = writer.Write(first)
	require.NoError(t, err)
	second := []byte(strings.Repeat("B", 60))
	_, err = writer.Write(second)
	var rotateErr *RotateError
	require.ErrorAs(t, err, &rotateErr)
	assert.True(t, writer.rotateRetryAt.After(time.Now()), "retry is scheduled in the future")

	// Renaming works again, but the retry interval has not passed yet
	renameFile = os.Rename
	_, err = writer.Write([]byte("C"))
	require.NoError(t, err)
	files, err := listDir(dir, "test.log*")
	require.NoError(t, err)
	require.Len(t, files, 1, "no rotation within the retry interval")

	// Pretend the retry interval has passed
	writer.rotateRetryAt = time.Now()
	last := []byte("after rotation\n")
	n, err := writer.Write(last)
	require.NoError(t, err)
	assert.Equal(t, len(last), n)
	require.NoError(t, writer.Sync())

	files, err = listDir(dir, "test.log.*")
	require.NoError(t, err)
	require.Len(t, files, 1, "one rotated file after the retry")
	rotated, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Equal(t, string(first)+string(second)+"C", string(rotated))
	current, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, string(last), string(current))
}

// TestRotatingWriter_RecoveryAfterCloseError verifies that a failing Close
// during a rotation does not leave the writer with a closed file.
func TestRotatingWriter_RecoveryAfterCloseError(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 100)
	require.NoError(t, err)
	defer writer.Close()

	first := []byte(strings.Repeat("A", 60))
	_, err = writer.Write(first)
	require.NoError(t, err)

	// Closing the file behind the writer's back makes Close fail in rotate
	require.NoError(t, writer.file.Close())

	second := []byte(strings.Repeat("B", 60))
	n, err := writer.Write(second)
	var rotateErr *RotateError
	require.ErrorAs(t, err, &rotateErr)
	assert.ErrorIs(t, err, os.ErrClosed)
	assert.Equal(t, len(second), n, "data is written to the opened file again")

	third := []byte("third\n")
	_, err = writer.Write(third)
	require.NoError(t, err)
	require.NoError(t, writer.Sync())

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, string(first)+string(second)+string(third), string(content))
}

// TestRotatingWriter_RecoveryAfterOpenError verifies that the writer
// opens the log file again with the next write when opening it
// after a rotation failed.
func TestRotatingWriter_RecoveryAfterOpenError(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 100)
	require.NoError(t, err)
	defer writer.Close()

	_, err = writer.Write([]byte(strings.Repeat("A", 60)))
	require.NoError(t, err)

	// After renaming, a directory at the log file path makes opening it fail
	renameFile = func(oldpath, newpath string) error {
		if err := os.Rename(oldpath, newpath); err != nil {
			return err
		}
		return os.Mkdir(oldpath, 0o755)
	}
	t.Cleanup(func() { renameFile = os.Rename })

	lost := []byte(strings.Repeat("B", 60))
	n, err := writer.Write(lost)
	require.Error(t, err)
	_, isRotateErr := errors.AsType[*RotateError](err)
	assert.False(t, isRotateErr, "the rotation succeeded, opening the file failed")
	assert.Equal(t, 0, n, "data could not be written")
	require.NoError(t, writer.Sync(), "Sync without an open file")

	// Opening keeps failing while the directory is there,
	// and without an open file no rotation is attempted
	n, err = writer.Write(lost)
	require.Error(t, err)
	_, isRotateErr = errors.AsType[*RotateError](err)
	assert.False(t, isRotateErr, "no rotation without an open file")
	assert.True(t, writer.rotateRetryAt.IsZero(), "no rotation failed, so no retry delay")
	assert.Equal(t, 0, n)

	// The next write opens the file once the cause is gone
	renameFile = os.Rename
	require.NoError(t, os.Remove(filePath))
	data := []byte("recovered\n")
	n, err = writer.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	require.NoError(t, writer.Sync())

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, string(data), string(content))
}

// TestRotatingWriter_WriteAfterCloseWithoutOpenFile verifies that Close
// prevents writes from opening the log file again after a rotation.
func TestRotatingWriter_WriteAfterCloseWithoutOpenFile(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 100)
	require.NoError(t, err)

	// Leave the writer without an open file, like a failed open after a rotation
	require.NoError(t, writer.file.Close())
	writer.file = nil

	require.NoError(t, writer.Close())
	n, err := writer.Write([]byte("after close\n"))
	assert.ErrorIs(t, err, os.ErrClosed)
	assert.Equal(t, 0, n)
	assert.Nil(t, writer.file, "Write after Close does not open the file")
}

// TestRotatingWriter_RotateErrorJoinedWithOpenError verifies that Write
// reports both a failed rotation and a failed open of the log file.
//
// Value: protects=RotateError joined with the open error when both fail; fails_when=joinRotateError drops either error or n>0 is returned; why_new=other tests fail only one step; seam=none; no_claim=joining with a failing file.Write
func TestRotatingWriter_RotateErrorJoinedWithOpenError(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 100)
	require.NoError(t, err)
	defer writer.Close()

	_, err = writer.Write([]byte(strings.Repeat("A", 60)))
	require.NoError(t, err)

	// Renaming fails and leaves a directory at the log file path,
	// so opening the file after the failed rotation fails too
	renameFile = func(oldpath, newpath string) error {
		if err := os.Remove(oldpath); err != nil {
			return err
		}
		if err := os.Mkdir(oldpath, 0o755); err != nil {
			return err
		}
		return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: os.ErrPermission}
	}
	t.Cleanup(func() { renameFile = os.Rename })

	n, err := writer.Write([]byte(strings.Repeat("B", 60)))
	assert.Equal(t, 0, n, "data could not be written")
	var rotateErr *RotateError
	require.ErrorAs(t, err, &rotateErr, "the rotation error is kept")
	assert.ErrorIs(t, rotateErr, os.ErrPermission)
	assert.NotEqual(t, error(rotateErr), err, "the open error is joined to the rotation error")
	assert.ErrorContains(t, err, "error opening rotating log file")
	assert.Contains(t, rotateErr.Error(), filePath)
}

// TestNewRotatingWriter_OpenError verifies that the constructors
// return the error of opening the log file and no writer.
func TestNewRotatingWriter_OpenError(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "missing-dir", "test.log")

	writer, err := NewRotatingWriter(filePath, "", 0644, 100)
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.ErrorContains(t, err, "error opening rotating log file")
	assert.Nil(t, writer)

	writer, err = NewRotatingWriterWithCallback(filePath, "", 0644, 100, func(string, []byte, int, error) {})
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.Nil(t, writer)
}

// TestNewRotatingWriter_StatError verifies that a log file whose size
// can't be read is closed again, so its file descriptor does not leak,
// and that the error is returned and no writer.
func TestNewRotatingWriter_StatError(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "test.log")

	statErr := errors.New("stat failed")
	var opened *os.File
	statFile = func(file *os.File) (os.FileInfo, error) {
		opened = file
		return nil, statErr
	}
	t.Cleanup(func() { statFile = (*os.File).Stat })

	writer, err := NewRotatingWriter(filePath, "", 0644, 100)
	assert.ErrorIs(t, err, statErr)
	assert.ErrorContains(t, err, "error getting size of rotating log file")
	assert.Nil(t, writer)
	require.NotNil(t, opened, "the file was opened before stat")
	assert.ErrorIs(t, opened.Close(), os.ErrClosed, "the file was closed after the stat error")
}

// TestRotatingWriter_StatErrorOnReopen verifies that Write returns
// the stat error of opening the log file after a rotation,
// keeps no file open, and recovers with the next Write.
func TestRotatingWriter_StatErrorOnReopen(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 100)
	require.NoError(t, err)
	defer writer.Close()

	_, err = writer.Write([]byte(strings.Repeat("A", 60)))
	require.NoError(t, err)

	statErr := errors.New("stat failed")
	statFile = func(*os.File) (os.FileInfo, error) { return nil, statErr }
	t.Cleanup(func() { statFile = (*os.File).Stat })

	// The rotation succeeds, opening the new file fails at stat
	n, err := writer.Write([]byte(strings.Repeat("B", 60)))
	assert.ErrorIs(t, err, statErr)
	_, isRotateErr := errors.AsType[*RotateError](err)
	assert.False(t, isRotateErr, "the rotation succeeded")
	assert.Equal(t, 0, n)
	assert.Nil(t, writer.file, "the file is closed after the stat error")

	statFile = (*os.File).Stat
	data := []byte("recovered\n")
	n, err = writer.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, string(data), string(content))
}

// TestRotateError_Error verifies the error message of a RotateError
// and that it wraps the error of closing or renaming the log file.
func TestRotateError_Error(t *testing.T) {
	cause := errors.New("disk on fire")
	err := &RotateError{FilePath: "/var/log/app.log", Err: cause}
	assert.Equal(t, `error rotating log file "/var/log/app.log": disk on fire`, err.Error())
	assert.ErrorIs(t, err, cause)
	assert.Same(t, cause, errors.Unwrap(err))
}

// TestRotatingWriter_SyncError verifies that Sync returns
// the error of syncing the log file instead of hiding it.
func TestRotatingWriter_SyncError(t *testing.T) {
	writer, err := NewRotatingWriter(filepath.Join(t.TempDir(), "test.log"), "", 0644, 0)
	require.NoError(t, err)

	// Closing the file behind the writer's back makes syncing it fail
	require.NoError(t, writer.file.Close())
	assert.ErrorIs(t, writer.Sync(), os.ErrClosed)
}

// TestRotatingWriter_CloseError verifies that Close returns
// the error of closing the log file instead of hiding it.
func TestRotatingWriter_CloseError(t *testing.T) {
	writer, err := NewRotatingWriter(filepath.Join(t.TempDir(), "test.log"), "", 0644, 0)
	require.NoError(t, err)

	require.NoError(t, writer.Close())
	assert.ErrorIs(t, writer.Close(), os.ErrClosed, "the second Close closes the file again")
}

// recordErrorHandler replaces golog.ErrorHandler until the test ends
// and returns a function that returns the errors it received.
func recordErrorHandler(t *testing.T) (handledErrs func() []error) {
	t.Helper()
	var (
		mtx  sync.Mutex
		errs []error
	)
	prevHandler := golog.ErrorHandler()
	golog.SetErrorHandler(func(err error) {
		mtx.Lock()
		defer mtx.Unlock()
		errs = append(errs, err)
	})
	t.Cleanup(func() { golog.SetErrorHandler(prevHandler) })
	return func() []error {
		mtx.Lock()
		defer mtx.Unlock()
		return append([]error(nil), errs...)
	}
}

// TestRotatingWriter_WriteCallbackPanicKeepsResult verifies that
// a panicking callback never changes what Write returns or writes,
// whatever the panic value is and whether the write succeeded,
// failed, or rotated the file, and that the panic is reported
// with its value, the log file path and a stack trace.
func TestRotatingWriter_WriteCallbackPanicKeepsResult(t *testing.T) {
	data := []byte(strings.Repeat("B", 60))

	tests := []struct {
		name       string
		panicValue any
		prepare    func(t *testing.T, writer *RotatingWriter)
		wantN      int
		checkErr   func(t *testing.T, err error)
		wantPanic  string
		wantInFile bool
	}{
		{
			name:       "string panic after successful write",
			panicValue: "callback panic",
			wantN:      len(data),
			checkErr:   func(t *testing.T, err error) { assert.NoError(t, err) },
			wantPanic:  "callback panic",
			wantInFile: true,
		},
		{
			name:       "error panic after successful write",
			panicValue: errors.New("callback error"),
			wantN:      len(data),
			checkErr:   func(t *testing.T, err error) { assert.NoError(t, err) },
			wantPanic:  "callback error",
			wantInFile: true,
		},
		{
			name:       "nil panic is reported, not dropped",
			panicValue: nil,
			wantN:      len(data),
			checkErr:   func(t *testing.T, err error) { assert.NoError(t, err) },
			wantPanic:  "panic called with nil argument",
			wantInFile: true,
		},
		{
			name:       "panic after failed write keeps the write error",
			panicValue: "callback panic",
			prepare: func(t *testing.T, writer *RotatingWriter) {
				require.NoError(t, writer.Close())
			},
			wantN:     0,
			checkErr:  func(t *testing.T, err error) { assert.ErrorIs(t, err, os.ErrClosed) },
			wantPanic: "callback panic",
		},
		{
			name:       "panic after failed rotation keeps the RotateError",
			panicValue: "callback panic",
			prepare: func(t *testing.T, writer *RotatingWriter) {
				failRename(t)
				_, err := writer.Write([]byte(strings.Repeat("A", 60)))
				require.NoError(t, err)
			},
			wantN: len(data),
			checkErr: func(t *testing.T, err error) {
				var rotateErr *RotateError
				assert.ErrorAs(t, err, &rotateErr)
			},
			wantPanic:  "callback panic",
			wantInFile: true,
		},
		{
			name:       "panic after successful rotation",
			panicValue: "callback panic",
			prepare: func(t *testing.T, writer *RotatingWriter) {
				_, err := writer.Write([]byte(strings.Repeat("A", 60)))
				require.NoError(t, err)
			},
			wantN:      len(data),
			checkErr:   func(t *testing.T, err error) { assert.NoError(t, err) },
			wantPanic:  "callback panic",
			wantInFile: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handledErrs := recordErrorHandler(t)

			filePath := filepath.Join(t.TempDir(), "test.log")
			writer, err := NewRotatingWriter(filePath, "", 0644, 100)
			require.NoError(t, err)
			defer writer.Close()
			if tt.prepare != nil {
				tt.prepare(t, writer)
			}

			var calls int
			writer.SetWriteCallback(func(string, []byte, int, error) {
				calls++
				panic(tt.panicValue)
			})

			n, err := writer.Write(data)
			assert.Equal(t, tt.wantN, n)
			tt.checkErr(t, err)
			assert.Equal(t, 1, calls)

			errs := handledErrs()
			require.Len(t, errs, 1)
			assert.ErrorContains(t, errs[0], "recovered panic from WriteCallback")
			assert.ErrorContains(t, errs[0], tt.wantPanic)
			assert.ErrorContains(t, errs[0], filePath)
			assert.ErrorContains(t, errs[0], "goroutine ", "the report has a stack trace")

			if tt.wantInFile {
				content, err := os.ReadFile(filePath)
				require.NoError(t, err)
				assert.True(t, strings.HasSuffix(string(content), string(data)), "data is in the log file")
			}
		})
	}
}

// TestRotatingWriter_WriteCallbackPanicConcurrent verifies that
// callbacks panicking in many goroutines at once are each recovered
// and reported, and that no write is lost or fails because of them.
func TestRotatingWriter_WriteCallbackPanicConcurrent(t *testing.T) {
	handledErrs := recordErrorHandler(t)

	filePath := filepath.Join(t.TempDir(), "test.log")
	writer, err := NewRotatingWriter(filePath, "", 0644, 0)
	require.NoError(t, err)
	defer writer.Close()

	writer.SetWriteCallback(func(string, []byte, int, error) {
		panic("callback panic")
	})

	const numGoroutines = 10
	const numWrites = 50
	data := []byte("message\n")
	var wg sync.WaitGroup
	for range numGoroutines {
		wg.Go(func() {
			for range numWrites {
				n, err := writer.Write(data)
				assert.NoError(t, err)
				assert.Equal(t, len(data), n)
			}
		})
	}
	wg.Wait()

	assert.Len(t, handledErrs(), numGoroutines*numWrites)
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, strings.Repeat(string(data), numGoroutines*numWrites), string(content))
}

// TestNewRotatingWriter_CleansFilePath verifies that the log file path
// is cleaned, so FilePath, the callback and rotated file names
// don't carry redundant path elements.
func TestNewRotatingWriter_CleansFilePath(t *testing.T) {
	dir := t.TempDir()

	writer, err := NewRotatingWriter(filepath.Join(dir, "sub")+"/../test.log", "", 0644, 0)
	require.NoError(t, err)
	defer writer.Close()

	assert.Equal(t, filepath.Join(dir, "test.log"), writer.FilePath())
}

// TestRotatingWriter_FilePerm verifies that the log file
// and the new log file after a rotation are created
// with the permissions passed to NewRotatingWriter.
func TestRotatingWriter_FilePerm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows only supports the read-only permission bit")
	}
	// 0400 is not changed by any common umask,
	// and the creating Write can still write to the new file
	const filePerm os.FileMode = 0400

	filePath := filepath.Join(t.TempDir(), "test.log")
	writer, err := NewRotatingWriter(filePath, "", filePerm, 100)
	require.NoError(t, err)
	defer writer.Close()

	info, err := os.Stat(filePath)
	require.NoError(t, err)
	assert.Equal(t, filePerm, info.Mode().Perm(), "log file created by NewRotatingWriter")

	// The second write rotates the file and creates a new one
	_, err = writer.Write([]byte(strings.Repeat("A", 60)))
	require.NoError(t, err)
	_, err = writer.Write([]byte(strings.Repeat("B", 60)))
	require.NoError(t, err)
	rotated, err := listDir(filepath.Dir(filePath), "test.log.*")
	require.NoError(t, err)
	require.Len(t, rotated, 1, "the file was rotated")

	info, err = os.Stat(filePath)
	require.NoError(t, err)
	assert.Equal(t, filePerm, info.Mode().Perm(), "log file created after the rotation")
}
