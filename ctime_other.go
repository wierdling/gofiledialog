//go:build !windows && !darwin

package gofiledialog

import (
	"os"
	"time"
)

// supportsCreatedTime is false here: most non-Darwin Unixes (notably Linux
// via the standard syscall.Stat_t) don't expose a reliable file birth time,
// so the Date-created column is hidden by default on these platforms.
const supportsCreatedTime = false

func createdTime(info os.FileInfo) time.Time {
	return time.Time{}
}
