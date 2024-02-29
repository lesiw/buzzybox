package hive_test

import (
	"strings"
	"testing"

	"lesiw.io/buzzybox/hive"
)

func TestTrue(t *testing.T) {
	cmd := hive.Command("true")
	cmd.Stdout = &strings.Builder{}
	cmd.Stderr = &strings.Builder{}
	if got := hive.Command("true", "--help").Run(); got != 0 {
		t.Errorf("true returned %d, want 0", got)
	}
	if cmd.Stdout.(*strings.Builder).String() != "" {
		t.Error("stdout not empty")
	}
	if cmd.Stderr.(*strings.Builder).String() != "" {
		t.Error("stderr not empty")
	}
}

func TestTrueSwallowsArgv(t *testing.T) {
	cmd := hive.Command("true", "--help")
	cmd.Stdout = &strings.Builder{}
	cmd.Stderr = &strings.Builder{}
	if got := hive.Command("true", "--help").Run(); got != 0 {
		t.Errorf("true returned %d, want 0", got)
	}
	if cmd.Stdout.(*strings.Builder).String() != "" {
		t.Error("stdout not empty")
	}
	if cmd.Stderr.(*strings.Builder).String() != "" {
		t.Error("stderr not empty")
	}
}
