package hive_test

import (
	"testing"

	"lesiw.io/buzzybox/hive"
)

func TestTrue(t *testing.T) {
	var (
		cmd            = hive.Command("true")
		outbuf, errbuf = new(syncBuffer), new(syncBuffer)
	)
	cmd.Stdout, cmd.Stderr = outbuf, errbuf
	if got := hive.Command("true", "--help").Run(); got != 0 {
		t.Errorf("true returned %d, want 0", got)
	}
	if outbuf.String() != "" {
		t.Error("stdout not empty")
	}
	if errbuf.String() != "" {
		t.Error("stderr not empty")
	}
}

func TestTrueSwallowsArgv(t *testing.T) {
	var (
		cmd            = hive.Command("true", "--help")
		outbuf, errbuf = new(syncBuffer), new(syncBuffer)
	)
	cmd.Stdout, cmd.Stderr = outbuf, errbuf
	if got := hive.Command("true", "--help").Run(); got != 0 {
		t.Errorf("true returned %d, want 0", got)
	}
	if outbuf.String() != "" {
		t.Error("stdout not empty")
	}
	if errbuf.String() != "" {
		t.Error("stderr not empty")
	}
}
