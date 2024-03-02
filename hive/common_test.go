package hive_test

import (
	"io"
	"os"
	"strings"
	"testing"
)

type strset map[string]bool

func stringset(s ...string) strset {
	m := make(strset)
	for _, k := range s {
		m[k] = true
	}
	return m
}

func intptr(i int) *int {
	return &i
}

func tmpfile(t *testing.T, contents string) string {
	tmpfile, err := os.CreateTemp("", "buzzybox-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(tmpfile.Name()); err != nil {
			t.Fatal(err)
		}
	})
	if _, err := tmpfile.WriteString(contents); err != nil {
		t.Fatal(err)
	}
	return tmpfile.Name()
}

type multiStringReader struct {
	current io.Reader
	strings []string
	offset  int
}

func newMultiStringReader(s []string) *multiStringReader {
	return &multiStringReader{strings: s}
}

func (r *multiStringReader) Read(p []byte) (n int, err error) {
	if r.current == nil {
		if r.offset >= len(r.strings) {
			return 0, io.EOF
		}
		r.current = strings.NewReader(r.strings[r.offset])
		r.offset++
	}
	n, err = r.current.Read(p)
	if err == io.EOF {
		r.current = nil
	}
	return
}
