package apps_test

import (
	"io"
	"strings"
	"testing"

	"lesiw.io/gobox/apps"
)

func run(t *testing.T, appFunc func([]string, *apps.IOs) int, argv ...string) string {
	outw := &strings.Builder{}
	ios := &apps.IOs{
		Out: outw,
		Err: io.Discard,
	}

	code := appFunc(argv, ios)
	if code != 0 {
		t.Errorf("code: got %d, want 0", code)
	}

	out := outw.String()
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("out: missing trailing newline")
	}
	out = strings.TrimSuffix(out, "\n")
	return out
}

func runN(t *testing.T, appFunc func([]string, *apps.IOs) int, argv ...string) []string {
	outw := &strings.Builder{}
	ios := &apps.IOs{
		Out: outw,
		Err: io.Discard,
	}

	code := appFunc(argv, ios)
	if code != 0 {
		t.Errorf("code: got %d, want 0", code)
	}

	out := outw.String()
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("missing trailing newline")
	}
	return strings.Split(strings.TrimSuffix(out, "\n"), "\n")
}

func fail(t *testing.T, appFunc func([]string, *apps.IOs) int, argv ...string) string {
	errw := &strings.Builder{}
	ios := &apps.IOs{
		Out: io.Discard,
		Err: errw,
	}

	code := appFunc(argv, ios)
	if code != 1 {
		t.Errorf("code: got %d, want 1", code)
	}

	err := errw.String()
	if !strings.HasSuffix(err, "\n") {
		t.Errorf("err: missing trailing newline")
	}
	err = strings.TrimSuffix(err, "\n")
	return err
}

func failN(t *testing.T, appFunc func([]string, *apps.IOs) int, argv ...string) []string {
	errw := &strings.Builder{}
	ios := &apps.IOs{
		Out: io.Discard,
		Err: errw,
	}

	code := appFunc(argv, ios)
	if code != 1 {
		t.Errorf("code: got %d, want 1", code)
	}

	err := errw.String()
	if !strings.HasSuffix(err, "\n") {
		t.Errorf("err: missing trailing newline")
	}
	return strings.Split(strings.TrimSuffix(err, "\n"), "\n")
}
