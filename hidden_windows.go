//go:build windows

package gofiledialog

import (
	"os"
	"syscall"
)

func isHidden(info os.FileInfo) bool {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return false
	}
	return data.FileAttributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0
}

// Windows visibility is an attribute of the target metadata, not its name.
func isHiddenName(_ string, info os.FileInfo) bool { return isHidden(info) }
