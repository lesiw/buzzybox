package hive

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
)

var basenameUsage = `usage: basename [-a] [-s SUFFIX] NAME... | NAME [SUFFIX]

Return non-directory portion of a pathname removing suffix.
`

func init() {
	Bees["basename"] = Basename
}

func Basename(cmd *Cmd) int {
	var (
		names    []string
		flags    = flag.NewFlagSet("basename", flag.ContinueOnError)
		allNames = flags.Bool("a", false, "All arguments are names")
		suffix   = flags.String("s", "", "Remove `suffix` (implies -a)")
	)
	flags.SetOutput(cmd.Stderr)
	flags.Usage = func() { fmt.Fprintln(flags.Output(), basenameUsage); flags.PrintDefaults() }

	if err := flags.Parse(cmd.Args[1:]); err != nil || flags.NArg() == 0 {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		} else if err == nil {
			fmt.Fprintln(cmd.Stderr, "error: needs 1 argument")
			flags.Usage()
		}
		return 1
	}

	if *allNames || *suffix != "" {
		names = flags.Args()
	} else if len(flags.Args()) > 2 {
		fmt.Fprintln(cmd.Stderr, "error: too many arguments")
		return 1
	} else {
		*suffix = flags.Arg(1)
		names = flags.Args()[:1]
	}

	for _, name := range names {
		name = filepath.Base(name)
		if *suffix != "" {
			name = strings.TrimSuffix(name, *suffix)
		}
		fmt.Fprintln(cmd.Stdout, name)
	}

	return 0
}
