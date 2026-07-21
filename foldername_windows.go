//go:build windows

package gofiledialog

import (
	"errors"
	"strings"
)

var reservedWindowsFolderNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

func validatePlatformFolderName(name string) error {
	if strings.ContainsAny(name, `<>:"|?*`) {
		return errors.New("folder name contains an invalid character")
	}
	if strings.HasSuffix(name, " ") || strings.HasSuffix(name, ".") {
		return errors.New("folder name must not end with a space or period")
	}
	root := name
	if dot := strings.IndexByte(root, '.'); dot >= 0 {
		root = root[:dot]
	}
	if reservedWindowsFolderNames[strings.ToUpper(root)] {
		return errors.New("folder name is reserved on this platform")
	}
	return nil
}
