package hive_test

import (
	"bytes"
	"sync"
)

// syncBuffer is a bytes.Buffer safe for concurrent writes. The real
// os.Stdout and os.Stderr are safe to write from multiple goroutines
// (each write is an independent syscall), but a bytes.Buffer is not,
// and awk's pipe output streams into the shared sink concurrently with
// its own prints. Tests use syncBuffer in place of a bare buffer so the
// race detector stays quiet.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *syncBuffer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *syncBuffer) WriteString(s string) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.WriteString(s)
}

func (w *syncBuffer) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}
