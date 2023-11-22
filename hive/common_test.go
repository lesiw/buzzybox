package hive_test

type strset map[string]bool

func stringset(s ...string) strset {
	m := make(strset)
	for _, k := range s {
		m[k] = true
	}
	return m
}
