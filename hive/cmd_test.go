package hive_test

import (
	"strings"
	"testing"

	"lesiw.io/buzzybox/hive"
)

func run(t *testing.T, argv ...string) string {
	outw := &strings.Builder{}
	cmd := hive.Command(argv...)
	cmd.Stdout = outw
	if code := cmd.Run(); code != 0 {
		t.Errorf("code: got %d, want 0", code)
	}
	out := outw.String()
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("out: missing trailing newline")
	}
	return strings.TrimSuffix(out, "\n")
}

func runN(t *testing.T, argv ...string) []string {
	return strings.Split(run(t, argv...), "\n")
}

func fail(t *testing.T, argv ...string) string {
	errw := &strings.Builder{}
	cmd := hive.Command(argv...)
	cmd.Stderr = errw
	if code := cmd.Run(); code != 1 {
		t.Errorf("code: got %d, want 1", code)
	}
	err := errw.String()
	if !strings.HasSuffix(err, "\n") {
		t.Errorf("err: missing trailing newline")
	}
	return strings.TrimSuffix(err, "\n")
}

func failN(t *testing.T, argv ...string) []string {
	return strings.Split(fail(t, argv...), "\n")
}
