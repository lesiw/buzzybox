package apps_test

import (
	"testing"

	"lesiw.io/gobox/apps"
)

func TestFalse(t *testing.T) {
	got := apps.False([]string{}, nil)
	if got != 1 {
		t.Errorf("False() = %d; want 1", got)
	}
}

func TestFalseSwallowsArgv(t *testing.T) {
	got := apps.False([]string{"-h"}, nil)
	if got != 1 {
		t.Errorf("False() = %d; want 1", got)
	}
}
