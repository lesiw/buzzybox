package hive_test

import (
	"testing"

	"lesiw.io/buzzybox/hive"
)

func TestFalse(t *testing.T) {
	if got := hive.Command("false").Run(); got != 1 {
		t.Errorf("false returned %d, want 1", got)
	}
}

func TestFalseSwallowsArgv(t *testing.T) {
	if got := hive.Command("false", "-h").Run(); got != 1 {
		t.Errorf("false returned %d, want 1", got)
	}
}
