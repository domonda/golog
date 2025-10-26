package logfile

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ungerik/go-fs"

	"github.com/domonda/golog"
	"github.com/domonda/golog/log"
)

func TestRotatingWriter_BasicWrite(t *testing.T) {
	dir := fs.MustMakeTempDir()
	t.Cleanup(func() {
		dir.RemoveRecursive()
	})

	filePath := dir.Join("test.log").LocalPath()
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
	dir := fs.MustMakeTempDir()
	t.Cleanup(func() {
		dir.RemoveRecursive()
	})

	filePath := dir.Join("test.log").LocalPath()
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
	files, err := dir.ListDirMax(-1, "test.log*")
	require.NoError(t, err)
	assert.Equal(t, 1, len(files), "should only have one file with no rotation")
}

func TestRotatingWriter_SingleRotation(t *testing.T) {
	dir := fs.MustMakeTempDir()
	t.Cleanup(func() {
		dir.RemoveRecursive()
	})

	filePath := dir.Join("test.log").LocalPath()
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
	files, err := dir.ListDirMax(-1, "test.log*")
	require.NoError(t, err)
	assert.Equal(t, 2, len(files), "should have original and one rotated file")

	// Verify the current file contains only the second write
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, string(data2), string(content))
}

func TestRotatingWriter_MultipleRotations(t *testing.T) {
	dir := fs.MustMakeTempDir()
	t.Cleanup(func() {
		dir.RemoveRecursive()
	})

	filePath := dir.Join("test.log").LocalPath()
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
	files, err := dir.ListDirMax(-1, "test.log*")
	require.NoError(t, err)
	expectedFiles := numWrites + 1 // Current file + one rotated file per write after first
	assert.Equal(t, expectedFiles, len(files), "should have %d files after %d writes", expectedFiles, numWrites)
}

func TestRotatingWriter_CustomTimeFormat(t *testing.T) {
	dir := fs.MustMakeTempDir()
	t.Cleanup(func() {
		dir.RemoveRecursive()
	})

	filePath := dir.Join("test.log").LocalPath()
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

	// Verify rotated file uses custom format
	files, err := dir.ListDirMax(-1, "test.log.*")
	require.NoError(t, err)
	require.Greater(t, len(files), 0, "should have at least one rotated file")

	// Check that the rotated file name contains date in expected format
	rotatedFile := files[0].Name()
	assert.Contains(t, rotatedFile, "test.log.")
	// Should match pattern like: test.log.2024-01-15
	parts := strings.Split(rotatedFile, ".")
	require.Greater(t, len(parts), 2, "rotated file should have date suffix")
}

func TestRotatingWriter_ExistingFile(t *testing.T) {
	dir := fs.MustMakeTempDir()
	t.Cleanup(func() {
		dir.RemoveRecursive()
	})

	filePath := dir.Join("existing.log").LocalPath()

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
	dir := fs.MustMakeTempDir()
	t.Cleanup(func() {
		dir.RemoveRecursive()
	})

	filePath := dir.Join("test.log").LocalPath()
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
	dir := fs.MustMakeTempDir()
	t.Cleanup(func() {
		dir.RemoveRecursive()
	})

	filePath := dir.Join("test.log").LocalPath()
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
	dir := fs.MustMakeTempDir()
	t.Cleanup(func() {
		dir.RemoveRecursive()
	})

	filePath := dir.Join("concurrent.log").LocalPath()
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
	files, err := dir.ListDirMax(-1, "concurrent.log*")
	require.NoError(t, err)
	assert.Greater(t, len(files), 1, "should have rotated at least once with concurrent writes")
}

// TestRotatingWriter_WithGolog tests integration with golog using concurrent writes
// This is the original test, now improved with better assertions
func TestRotatingWriter_WithGolog(t *testing.T) {
	dir := fs.MustMakeTempDir()
	t.Cleanup(func() {
		dir.RemoveRecursive()
	})

	const jsonRotateSize = 350 // fit two 165 byte lines, but not three
	jsonWriter, err := NewRotatingWriter(
		dir.Join("json.log").LocalPath(), "", 0644, jsonRotateSize)
	require.NoError(t, err)
	defer jsonWriter.Close()

	const textRotateSize = 300 // fit two 130 byte lines but not three
	textWriter, err := NewRotatingWriter(
		dir.Join("text.log").LocalPath(), "", 0644, textRotateSize)
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
	jsonFiles, err := dir.ListDirMax(-1, "json.log*")
	require.NoError(t, err)
	require.Equal(t, numFilesExpected, len(jsonFiles), "expected %d files for json.log*, got %d", numFilesExpected, len(jsonFiles))

	// Verify text log rotation
	textFiles, err := dir.ListDirMax(-1, "text.log*")
	require.NoError(t, err)
	require.Equal(t, numFilesExpected, len(textFiles), "expected %d files for text.log*, got %d", numFilesExpected, len(textFiles))

	// Verify that all log messages were written
	totalMessages := numThreads * numThreadMessages
	// Count messages in JSON files
	jsonMessageCount := 0
	for _, file := range jsonFiles {
		content, err := os.ReadFile(file.LocalPath())
		require.NoError(t, err)
		// Each line is a JSON log message
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		jsonMessageCount += len(lines)
	}
	assert.Equal(t, totalMessages, jsonMessageCount, "should have all messages in JSON files")
}
