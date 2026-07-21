//go:build !windows

package gofiledialog

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func lockFile(file *os.File, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return nil
		}
		if err != unix.EWOULDBLOCK && err != unix.EAGAIN {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for settings lock %s", file.Name())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func unlockFile(file *os.File) error { return unix.Flock(int(file.Fd()), unix.LOCK_UN) }
