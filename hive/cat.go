package hive

import (
	"bufio"
	"io"
	"os"

	"lesiw.io/buzzybox/internal/flag"
)

const catUsage = `usage: cat [-u] [FILE...]

Copy (concatenate) files to stdout. If no files are given, copy from stdin.
File "-" is a synonym for stdin.`

func init() {
	Bees["cat"] = Cat
}

func Cat(cmd *Cmd) int {
	var (
		err   error
		flags = flag.NewFlagSet(cmd.Stderr, "cat")
		unbuf = flags.Bool("u", "Disable output buffering.")
	)
	flags.Usage = catUsage
	if err := flags.Parse(cmd.Args[1:]...); err != nil {
		return 1
	}
	w := cmd.Stdout
	if *unbuf {
		w = bufio.NewWriter(w)
	}
	files := flags.Args
	if len(files) < 1 {
		files = []string{"-"}
	}
	var r io.Reader
	var file *os.File
	for _, f := range files {
		file = nil
		if f == "-" {
			r = cmd.Stdin
		} else {
			file, err = os.Open(f)
			if err != nil {
				cmd.Errorf("bad file: %v\n", err)
				return 1
			}
			r = file
		}
		if _, err := io.Copy(w, r); err != nil {
			cmd.Errorf("bad file: %v\n", err)
			return 1
		}
		if file != nil {
			if err := file.Close(); err != nil {
				cmd.Errorf("bad file: %v\n", err)
				return 1
			}
		}
	}
	if *unbuf {
		if err := w.(*bufio.Writer).Flush(); err != nil {
			cmd.Errorf("bad file: %v\n", err)
			return 1
		}
	}
	return 0
}
