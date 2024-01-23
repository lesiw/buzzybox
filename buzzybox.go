package main

import (
	"os"

	"lesiw.io/buzzybox/hive"
)

func main() {
	os.Exit(hive.Command(os.Args...).Run())
}
