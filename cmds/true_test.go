package gobox_test

import (
	"testing"

	gobox "lesiw.io/gobox/cmds"
)

func TestTrue(t *testing.T) {
	if got := gobox.Command("true").Run(); got != 0 {
		t.Errorf("true returned %d, want 0", got)
	}
}

func TestTrueSwallowsArgv(t *testing.T) {
	if got := gobox.Command("true", "-h").Run(); got != 0 {
		t.Errorf("true returned %d, want 0", got)
	}
}
