//go:build windows && !tinygo
// +build windows,!tinygo

package hive

import (
	"fmt"
)

func arch() string {
	info := sysinfo()
	switch info.wProcessorArchitecture {
	case PROCESSOR_ARCHITECTURE_AMD64:
		return "x86_64"
	case PROCESSOR_ARCHITECTURE_INTEL:
		if info.wProcessorLevel <= 3 {
			return "i386"
		} else if info.wProcessorLevel >= 6 {
			return "i686"
		} else {
			return fmt.Sprintf("i%d86", info.wProcessorLevel)
		}
	case PROCESSOR_ARCHITECTURE_ARM:
		return "arm"
	case PROCESSOR_ARCHITECTURE_ARM64:
		return "arm64"
	default:
		return ""
	}
}
