package hive_test

import (
	"strings"
	"testing"

	"lesiw.io/buzzybox/hive"
)

func TestFalse(t *testing.T) {
	cmd := hive.Command("false")
	cmd.Stdout = &strings.Builder{}
	cmd.Stderr = &strings.Builder{}
	if got := hive.Command("false", "--help").Run(); got != 1 {
		t.Errorf("false returned %d, want 1", got)
	}
	if cmd.Stdout.(*strings.Builder).String() != "" {
		t.Error("stdout not empty")
	}
	if cmd.Stderr.(*strings.Builder).String() != "" {
		t.Error("stderr not empty")
	}
}

func TestFalseSwallowsArgv(t *testing.T) {
	cmd := hive.Command("false", "--help")
	cmd.Stdout = &strings.Builder{}
	cmd.Stderr = &strings.Builder{}
	if got := hive.Command("false", "--help").Run(); got != 1 {
		t.Errorf("false returned %d, want 1", got)
	}
	if cmd.Stdout.(*strings.Builder).String() != "" {
		t.Error("stdout not empty")
	}
	if cmd.Stderr.(*strings.Builder).String() != "" {
		t.Error("stderr not empty")
	}
}
