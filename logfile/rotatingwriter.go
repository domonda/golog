package logfile

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

var RotatingWriterTimeFormat = "2006-01-02_15:04:05"

// RotatingWriter implements io.WriteCloser
type RotatingWriter struct {
	mtx sync.Mutex

	filePath   string
	filePerm   os.FileMode
	file       *os.File
	size       int64
	rotateSize int64
}

func NewRotatingWriter(filePath string, filePerm os.FileMode, rotateSize int64) (*RotatingWriter, error) {
	filePath = filepath.Clean(filePath)
	file, size, err := openFile(filePath, filePerm)
	if err != nil {
		return nil, err
	}
	rw := &RotatingWriter{
		filePath:   filePath,
		filePerm:   filePerm,
		file:       file,
		size:       size,
		rotateSize: rotateSize,
	}
	return rw, nil
}

func openFile(filePath string, filePerm os.FileMode) (file *os.File, size int64, err error) {
	file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm) //#nosec G304
	if err != nil {
		return nil, 0, err
	}
	stat, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	return file, stat.Size(), nil
}

func (rw *RotatingWriter) FilePath() string {
	return rw.filePath
}

func (rw *RotatingWriter) RotateSize() int64 {
	return rw.rotateSize
}

func (rw *RotatingWriter) Write(msg []byte) (n int, err error) {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	rw.size += int64(len(msg))
	if rw.rotateSize > 0 && rw.size >= rw.rotateSize {
		err := rw.rotate()
		if err != nil {
			return 0, err
		}
		rw.size += int64(len(msg))
	}

	return rw.file.Write(msg)
}

func (rw *RotatingWriter) rotate() error {
	err := rw.file.Close()
	if err != nil {
		return err
	}

	err = os.Rename(rw.filePath, rw.rotatedFilePath())
	if err != nil {
		return err
	}

	file, size, err := openFile(rw.filePath, rw.filePerm)
	if err != nil {
		return err
	}
	rw.file = file
	rw.size = size
	return nil
}

func (rw *RotatingWriter) rotatedFilePath() string {
	rotatedBase := rw.filePath + "." + time.Now().Format(RotatingWriterTimeFormat)

	rotated := rotatedBase
	for i := 1; fileExists(rotated); i++ {
		rotated = rotatedBase + "." + strconv.Itoa(i)
	}

	return rotated
}

// Sync flushed the file by calling os.File.Sync.
func (rw *RotatingWriter) Sync() error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	return rw.file.Sync()
}

func (rw *RotatingWriter) Close() error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	return rw.file.Close()
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}
