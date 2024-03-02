package hive

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"lesiw.io/buzzybox/internal/bbio"
	"lesiw.io/buzzybox/internal/flag"
)

const base64Usage = `usage: base64 [-d] [-w COLUMNS] [FILE]

Encode or decode base64.`

func init() {
	Bees["base64"] = Base64
}

func Base64(cmd *Cmd) int {
	flags := flag.NewFlagSet(cmd.Stderr, "base64")
	decode := flags.Bool("d", "Decode")
	wrap := flags.Int("w", "Wrap output at `columns` (default 76, 0 to disable)")
	flags.Usage = base64Usage
	if err := flags.Parse(cmd.Args[1:]...); err != nil {
		return 1
	}
	var file io.Reader
	var err error
	switch len(flags.Args) {
	case 0:
		file = cmd.Stdin
	case 1:
		if file, err = os.Open(flags.Args[0]); err != nil {
			fmt.Fprintln(cmd.Stderr, err)
			return 1
		}
	default:
		flags.PrintError("bad argc: want 0 or 1")
		return 1
	}
	if !flags.Set("w") {
		*wrap = 76
	}
	if *decode {
		_, err := io.Copy(cmd.Stdout, base64.NewDecoder(base64.StdEncoding, file))
		if err != nil {
			fmt.Fprintln(cmd.Stderr, err)
			return 1
		}
	} else {
		w := bbio.NewWrapWriter(cmd.Stdout, *wrap)
		encoder := base64.NewEncoder(base64.StdEncoding, w)
		_, err := io.Copy(encoder, file)
		_ = encoder.Close()
		fmt.Fprintln(cmd.Stdout)
		if err != nil {
			fmt.Fprintln(cmd.Stderr, err)
			return 1
		}
	}
	return 0
}
