package logfile

import (
	"os"
	"sync"
	"time"
)

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
	if rotateSize > 0 && size >= rotateSize {
		err = rw.rotate()
		if err != nil {
			file.Close()
			return nil, err
		}
	}
	return rw, nil
}

func openFile(filePath string, filePerm os.FileMode) (file *os.File, size int64, err error) {
	file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm|os.ModeExclusive)
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

func (rw *RotatingWriter) Write(logLine []byte) (n int, err error) {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	n, err = rw.file.Write(logLine)
	if err != nil {
		return n, err
	}

	rw.size += int64(len(logLine))
	if rw.rotateSize > 0 && rw.size >= rw.rotateSize {
		err := rw.rotate()
		if err != nil {
			return n, err
		}
	}

	return n, nil
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
	return rw.filePath + "." + time.Now().Format("2006-01-02T15:04:05.99")
}

func (rw *RotatingWriter) Close() error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	return rw.file.Close()
}
