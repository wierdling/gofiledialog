//go:build darwin

package gofiledialog

import (
	"os"
	"syscall"
	"time"
)

// supportsCreatedTime reports whether createdTime can return a real value.
const supportsCreatedTime = true

func createdTime(info os.FileInfo) time.Time {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}
	}
	return time.Unix(stat.Birthtimespec.Unix())
}
