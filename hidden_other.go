//go:build !windows

package gofiledialog

import (
	"os"
	"strings"
)

func isHidden(info os.FileInfo) bool {
	return isHiddenName(info.Name(), info)
}

func isHiddenName(name string, _ os.FileInfo) bool { return strings.HasPrefix(name, ".") }
