package hive

func init() {
	Bees["false"] = False
}

func False(*Cmd) int {
	return 1
}
