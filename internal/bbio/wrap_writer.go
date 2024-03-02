package bbio

import "io"

type WrapWriter struct {
	w io.Writer
	c int
	n int
}

func NewWrapWriter(w io.Writer, c int) *WrapWriter {
	return &WrapWriter{w, c, 0}
}

func (ww *WrapWriter) Write(buf []byte) (n int, err error) {
	defer func() { ww.n += n }()
	if ww.c <= 0 {
		return ww.w.Write(buf)
	}
	for cl := 0; len(buf) > 0 && err == nil; {
		cl = (ww.n + n) % ww.c
		if cl+len(buf) > ww.c {
			len := ww.c - cl
			wbuf := make([]byte, len+1)
			copy(wbuf, buf[:len])
			wbuf[len] = '\n'
			_, err = ww.w.Write(wbuf)
			n += len
			buf = buf[len:]
		} else {
			_, err = ww.w.Write(buf)
			n += len(buf)
			buf = buf[:0]
		}
	}
	return
}
