package gobox_test

import (
	"testing"

	gobox "lesiw.io/gobox/cmds"
)

func TestFalse(t *testing.T) {
	if got := gobox.Command("false").Run(); got != 1 {
		t.Errorf("false returned %d, want 1", got)
	}
}

func TestFalseSwallowsArgv(t *testing.T) {
	if got := gobox.Command("false", "-h").Run(); got != 1 {
		t.Errorf("false returned %d, want 1", got)
	}
}
