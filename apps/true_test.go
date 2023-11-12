package apps_test

import (
	"testing"

	"lesiw.io/gobox/apps"
)

func TestTrue(t *testing.T) {
	got := apps.True([]string{}, nil)
	if got != 0 {
		t.Errorf("True() = %d; want 0", got)
	}
}

func TestTrueSwallowsArgv(t *testing.T) {
	got := apps.True([]string{"-h"}, nil)
	if got != 0 {
		t.Errorf("True() = %d; want 0", got)
	}
}
