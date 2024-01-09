package gobox

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Cmd struct {
	exec.Cmd
	ExitCode int
	Fallback bool
}

type CmdFunc func(*Cmd) int

var Cmds = map[string]CmdFunc{
	"basename": Basename,
	"false":    False,
	"true":     True,
}

func Command(argv ...string) *Cmd {
	c := &Cmd{}
	c.Path = argv[0]
	c.Args = argv
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

func (c *Cmd) Run() int {
	cmd := filepath.Base(c.Path)
	if cmd == "gobox" {
		if len(c.Args) < 2 {
			fmt.Fprintln(c.Stderr, "Usage: gobox [command]")
			fmt.Fprintln(c.Stderr, "Available commands:")
			for app := range Cmds {
				fmt.Fprintln(c.Stderr, app)
			}
			return 1
		}
		c.Args = c.Args[1:]
		c.Path = c.Args[0]
		cmd = filepath.Base(c.Path)
	}
	if fn, ok := Cmds[cmd]; ok {
		c.ExitCode = fn(c)
	} else if c.Fallback {
		var ee *exec.ExitError
		if err := c.Cmd.Run(); err != nil && !errors.As(err, &ee) {
			fmt.Fprintln(c.Stderr, "bad command:", c.Cmd.Args[0])
			c.ExitCode = 1
		} else {
			c.ExitCode = c.ProcessState.ExitCode()
		}
	} else {
		fmt.Fprintln(c.Stderr, "bad command:", c.Cmd.Args[0])
		c.ExitCode = 1
	}
	return c.ExitCode
}
