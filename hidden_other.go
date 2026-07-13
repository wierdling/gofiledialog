//go:build !windows

package gofiledialog

import (
	"os"
	"strings"
)

func isHidden(info os.FileInfo) bool {
	return strings.HasPrefix(info.Name(), ".")
}
