package hive_test

import (
	"os"
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
