package hive_test

import (
	"testing"

	"lesiw.io/buzzybox/hive"
)

func TestTrue(t *testing.T) {
	if got := hive.Command("true").Run(); got != 0 {
		t.Errorf("true returned %d, want 0", got)
	}
}

func TestTrueSwallowsArgv(t *testing.T) {
	if got := hive.Command("true", "-h").Run(); got != 0 {
		t.Errorf("true returned %d, want 0", got)
	}
}
