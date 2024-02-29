package hive

import (
	"fmt"
	"path/filepath"
	"strings"

	"lesiw.io/buzzybox/internal/flag"
)

const basenameUsage = `usage: basename [-a] [-s SUFFIX] NAME... | NAME [SUFFIX]

Return non-directory portion of a pathname removing suffix.`

func init() {
	Bees["basename"] = Basename
}

func Basename(cmd *Cmd) int {
	var (
		names    []string
		flags    = flag.NewFlagSet(cmd.Stderr, "basename")
		allNames = flags.Bool("a", "All arguments are names")
		suffix   = flags.String("s", "Remove `suffix` (implies -a)")
	)
	flags.Usage = basenameUsage
	if err := flags.Parse(cmd.Args[1:]...); err != nil || len(flags.Args) == 0 {
		if err == nil {
			flags.PrintError("error: needs 1 argument")
		}
		return 1
	}
	if *allNames || *suffix != "" {
		names = flags.Args
	} else if len(flags.Args) > 2 {
		fmt.Fprintln(cmd.Stderr, "error: too many arguments")
		return 1
	} else {
		*suffix = flags.Arg(1)
		names = flags.Args[:1]
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
