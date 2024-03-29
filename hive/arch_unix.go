//go:build !windows && !tinygo
// +build !windows,!tinygo

package hive

import "golang.org/x/sys/unix"

func arch() string {
	var uname unix.Utsname
	if err := unix.Uname(&uname); err != nil {
		return ""
	}
	return string(uname.Machine[:])
}
