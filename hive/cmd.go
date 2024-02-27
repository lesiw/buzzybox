package hive

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
)

type Cmd struct {
	exec.Cmd
	Id       int
	Parent   *Cmd
	ExitCode int
	Fallback bool
	code     chan int
}
type CmdFunc func(*Cmd) int
type cmdTable struct {
	next atomic.Uint64
	cmd  []*Cmd
}

var procs cmdTable
var Bees = map[string]CmdFunc{}

func Command(argv ...string) *Cmd {
	c := &Cmd{}
	c.Path = argv[0]
	c.Args = argv
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Id = int(procs.next.Add(1))
	c.code = make(chan int)
	procs.cmd = append(procs.cmd, c)
	return c
}

func CmdList() (cmds []string) {
	for cmd := range Bees {
		cmds = append(cmds, cmd)
	}
	sort.Strings(cmds)
	return
}

func (c *Cmd) Default() int {
	// TODO: word wrap
	fmt.Fprintf(c.Stderr, "Usage: buzzybox [command]\nCommands: %s\n",
		strings.Join(CmdList(), ", "))
	return 1
}

func (c *Cmd) Run() int {
	c.Start()
	if err := c.Wait(); err != nil {
		c.ExitCode = 1
	}
	return c.ExitCode
}

func (c *Cmd) Start() {
	cmd := filepath.Base(c.Path)
	if cmd == "buzzybox" {
		if len(c.Args) < 2 {
			go func() { c.code <- c.Default() }()
			return
		}
		c.Args = c.Args[1:]
		c.Path = c.Args[0]
		cmd = filepath.Base(c.Path)
	}
	if fn, ok := Bees[cmd]; ok {
		go func() { c.code <- fn(c) }()
		return
	} else if c.Fallback {
		path, err := exec.LookPath(c.Path)
		if err != nil {
			goto badcmd
		}
		c.Path = path
		if err = c.Cmd.Start(); err != nil {
			goto badcmd
		}
		return
	}
badcmd:
	fmt.Fprintln(c.Stderr, "bad command:", c.Cmd.Args[0])
	c.ExitCode = 1
}

func (c *Cmd) Wait() error {
	if c.Process == nil {
		c.ExitCode = <-c.code
		return nil
	}
	if err := c.Cmd.Wait(); err != nil {
		return err
	}
	c.ExitCode = c.ProcessState.ExitCode()
	return nil
}

func (c *Cmd) spawn(argv ...string) *Cmd {
	cmd := Command(argv...)
	cmd.Fallback = true
	cmd.Stdin = c.Stdin
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	cmd.Parent = c
	return cmd
}

func (c *Cmd) StdinCloser() (io.WriteCloser, error) {
	c.Stdin = nil
	wc, err := c.StdinPipe()
	if err != nil {
		return nil, err
	}
	return &CmdWriteCloser{c, wc}, nil
}

type CmdWriteCloser struct {
	cmd *Cmd
	wc  io.WriteCloser
}

func (cwc *CmdWriteCloser) Write(p []byte) (int, error) {
	return cwc.wc.Write(p)
}

func (cwc *CmdWriteCloser) Close() error {
	if err := cwc.wc.Close(); err != nil {
		return err
	}
	return cwc.cmd.Wait()
}

func (c *Cmd) StdoutCloser() (io.ReadCloser, error) {
	c.Stdout = nil
	rc, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}
	return &cmdReadCloser{c, rc}, nil
}

type cmdReadCloser struct {
	cmd *Cmd
	rc  io.ReadCloser
}

func (crc *cmdReadCloser) Read(p []byte) (int, error) {
	return crc.rc.Read(p)
}

func (crc *cmdReadCloser) Close() error {
	if err := crc.rc.Close(); err != nil {
		return err
	}
	return crc.cmd.Wait()
}
