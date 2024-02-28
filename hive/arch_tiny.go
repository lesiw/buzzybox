//go:build tinygo
// +build tinygo

package hive

import "runtime"

func arch() string {
	switch runtime.GOARCH {
	case "386":
		return "i386"
	case "amd64":
		return "x86_64"
	default:
		return runtime.GOARCH
	}
}
