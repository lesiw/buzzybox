package hive_test

import (
	"testing"

	"lesiw.io/buzzybox/hive"
)

func TestFalse(t *testing.T) {
	var (
		cmd            = hive.Command("false")
		outbuf, errbuf = new(syncBuffer), new(syncBuffer)
	)
	cmd.Stdout, cmd.Stderr = outbuf, errbuf
	if got := hive.Command("false", "--help").Run(); got != 1 {
		t.Errorf("false returned %d, want 1", got)
	}
	if outbuf.String() != "" {
		t.Error("stdout not empty")
	}
	if errbuf.String() != "" {
		t.Error("stderr not empty")
	}
}

func TestFalseSwallowsArgv(t *testing.T) {
	var (
		cmd            = hive.Command("false", "--help")
		outbuf, errbuf = new(syncBuffer), new(syncBuffer)
	)
	cmd.Stdout, cmd.Stderr = outbuf, errbuf
	if got := hive.Command("false", "--help").Run(); got != 1 {
		t.Errorf("false returned %d, want 1", got)
	}
	if outbuf.String() != "" {
		t.Error("stdout not empty")
	}
	if errbuf.String() != "" {
		t.Error("stderr not empty")
	}
}
