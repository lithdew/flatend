package flatend

import (
	"errors"
	"fmt"
	"io"
)

type pipeReader struct {
	*io.PipeReader
}

func (r *pipeReader) Read(buf []byte) (n int, err error) {
	n, err = r.PipeReader.Read(buf)
	if err != nil && errors.Is(err, io.ErrClosedPipe) {
		err = io.EOF
	}
	return n, err
}

type pipeWriter struct {
	*io.PipeWriter
}

func (w *pipeWriter) Write(buf []byte) (n int, err error) {
	n, err = w.PipeWriter.Write(buf)
	if err != nil && errors.Is(err, io.ErrClosedPipe) {
		err = fmt.Errorf("%s: %w", err, io.EOF)
	}
	return n, err
}

// createWrappedPipe wraps around a reader/writer pair from io.Pipe() such that all
// errors reported by such reader/writer pair that comprise of io.ErrClosedPipe
// will be wrapped with io.EOF.
func createWrappedPipe() (*pipeReader, *pipeWriter) {
	r, w := io.Pipe()
	pr := &pipeReader{PipeReader: r}
	pw := &pipeWriter{PipeWriter: w}
	return pr, pw
}
