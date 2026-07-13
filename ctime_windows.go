//go:build windows

package gofiledialog

import (
	"os"
	"syscall"
	"time"
)

// supportsCreatedTime reports whether createdTime can return a real value.
const supportsCreatedTime = true

func createdTime(info os.FileInfo) time.Time {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return time.Time{}
	}
	return time.Unix(0, data.CreationTime.Nanoseconds())
}
