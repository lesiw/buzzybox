package main

import (
	"os"

	gobox "lesiw.io/gobox/cmds"
)

func main() {
	os.Exit(gobox.Command(os.Args...).Run())
}
