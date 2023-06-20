package logfile

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/ungerik/go-fs"

	"github.com/domonda/golog"
	"github.com/domonda/golog/log"
)

func TestRotatingWriter(t *testing.T) {
	dir := fs.MustMakeTempDir()

	const jsonRotateSize = 350 // fit two 165 byte lines, but not three
	jsonWriter, err := NewRotatingWriter(dir.Join("json.log").LocalPath(), 0600, jsonRotateSize)
	assert.NoError(t, err)
	defer jsonWriter.Close()

	const textRotateSize = 300 // fit two 130 byte lines but not three
	textWriter, err := NewRotatingWriter(dir.Join("text.log").LocalPath(), 0600, textRotateSize)
	assert.NoError(t, err)
	defer textWriter.Close()

	log.Config = golog.NewConfig(
		log.Levels,
		golog.AllLevelsActive,
		golog.NewJSONWriter(jsonWriter, &log.Format),
		golog.NewTextWriter(textWriter, &log.Format, golog.NoColorizer),
	)

	numThreads := 8
	numThreadMessages := 66

	var wg sync.WaitGroup
	wg.Add(numThreads)

	for i := 0; i < numThreads; i++ {
		go func(thread int) {
			for threadMsg := 0; threadMsg < numThreadMessages; threadMsg++ {
				log.Info("Thread log").
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

	numMessagesPerFile := 2
	numFilesExpected := numThreads * numThreadMessages / numMessagesPerFile

	files, err := dir.ListDirMax(-1, "json.log*")
	assert.NoError(t, err)
	assert.Equal(t, numFilesExpected, len(files), "expected %d files for json.log*, got %d", numFilesExpected, len(files))

	// t.Fatal(dir)
	if !t.Failed() {
		dir.RemoveRecursive()
	}
}
