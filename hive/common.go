package hive

import (
	"bufio"
	"io"
	"strings"
)

type prettyError interface {
	Error() string
	Pretty() string
}

type stringlist []string

func (s *stringlist) String() string {
	return "[" + strings.Join(*s, ", ") + "]"
}

func (s *stringlist) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func prettyPrintError(w io.Writer, err error) {
	if pe, ok := err.(prettyError); ok {
		_, _ = io.WriteString(w, pe.Pretty())
		_, _ = io.WriteString(w, "\n")
	} else {
		_, _ = io.WriteString(w, err.Error())
	}
}

type strset map[string]bool

func stringset(s ...string) strset {
	m := make(strset)
	for _, k := range s {
		m[k] = true
	}
	return m
}

func runealpha(r rune) bool {
	return 'A' <= r && r <= 'Z' || 'a' <= r && r <= 'z'
}

type runeScanCloser interface {
	io.Closer
	io.RuneScanner
}

type bufferedReadCloser struct {
	*bufio.Reader
	closer io.Closer
}

func newBufferedReadCloser(rc io.ReadCloser) *bufferedReadCloser {
	return &bufferedReadCloser{
		Reader: bufio.NewReader(rc),
		closer: rc,
	}
}

func (brc *bufferedReadCloser) Close() error {
	return brc.closer.Close()
}

func skiprune(reader io.RuneScanner, s rune) error {
	for {
		if r, _, err := reader.ReadRune(); err != nil {
			return err
		} else if r != s {
			_ = reader.UnreadRune()
			return nil
		}
	}
}
