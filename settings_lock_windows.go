//go:build windows

package gofiledialog

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

func lockFile(file *os.File, timeout time.Duration) error {
	overlapped := new(windows.Overlapped)
	deadline := time.Now().Add(timeout)
	for {
		err := windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, overlapped)
		if err == nil {
			return nil
		}
		if err != windows.ERROR_LOCK_VIOLATION {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for settings lock %s", file.Name())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func unlockFile(file *os.File) error {
	return windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, new(windows.Overlapped))
}
