package hive

import (
	"fmt"

	"lesiw.io/buzzybox/internal/flag"
)

const archUsage = `usage: arch

Print machine architecture.`

func init() {
	Bees["arch"] = Arch
}

func Arch(cmd *Cmd) int {
	flags := flag.NewFlagSet(cmd.Stderr, "arch")
	flags.Usage = archUsage
	if err := flags.Parse(cmd.Args[1:]...); err != nil {
		return 1
	}
	fmt.Fprintln(cmd.Stdout, arch())
	return 0
}
