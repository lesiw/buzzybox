//go:build plan9

package hive

import "runtime"

func arch() string {
	return runtime.GOARCH
}
