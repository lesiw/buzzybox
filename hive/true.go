package hive

func init() {
	Bees["true"] = True
}

func True(*Cmd) int {
	return 0
}
