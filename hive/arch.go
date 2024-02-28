package hive

import "fmt"

func init() {
	Bees["arch"] = Arch
}

func Arch(cmd *Cmd) int {
	fmt.Fprintln(cmd.Stdout, arch())
	return 0
}
