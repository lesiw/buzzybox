package hive

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
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

func CmdList() (cmds []string) {
	for cmd := range Cmds {
		cmds = append(cmds, cmd)
	}
	sort.Strings(cmds)
	return
}

func (c *Cmd) Run() int {
	c.ExitCode = c.run()
	return c.ExitCode
}

func (c *Cmd) run() int {
	cmd := filepath.Base(c.Path)
	if cmd == "buzzybox" {
		if len(c.Args) < 2 {
			// TODO: word wrap
			fmt.Fprintf(c.Stderr, "Usage: buzzybox [command]\nCommands: %s\n",
				strings.Join(CmdList(), ", "))
			return 1
		}
		c.Args = c.Args[1:]
		c.Path = c.Args[0]
		cmd = filepath.Base(c.Path)
	}
	if fn, ok := Cmds[cmd]; ok {
		return fn(c)
	} else if c.Fallback {
		path, err := exec.LookPath(c.Path)
		if err != nil {
			goto badcmd
		}
		c.Path = path
		var ee *exec.ExitError
		if err = c.Cmd.Run(); err != nil && !errors.As(err, &ee) {
			goto badcmd
		}
		return c.ProcessState.ExitCode()
	}
badcmd:
	fmt.Fprintln(c.Stderr, "bad command:", c.Cmd.Args[0])
	return 1
}
