package secret

import (
	"errors"
	"io"
)

type CountingWriter struct {
	writer       io.Writer
	BytesWritten int
}

func NewCountingWriter(writer io.Writer) *CountingWriter {
	return &CountingWriter{
		writer:       writer,
		BytesWritten: 0,
	}
}

func (w *CountingWriter) Write(p []byte) (n int, err error) {
	n, err = w.writer.Write(p)
	w.BytesWritten += n
	return
}

func NewHardLimitWriter(writer io.Writer, hardLimit int) *HardLimitWriter {
	return &HardLimitWriter{
		CountingWriter: NewCountingWriter(writer),
		hardLimit:      hardLimit,
	}
}

type HardLimitWriter struct {
	*CountingWriter
	hardLimit int
}

var ErrHardLimitReached = errors.New("hard read limit reached")

func (w *HardLimitWriter) Write(p []byte) (n int, err error) {
	// TODO: this check sucks
	n, err = w.CountingWriter.Write(p)
	if w.CountingWriter.BytesWritten > w.hardLimit {
		return 0, ErrHardLimitReached
	}
	return n, err
}
