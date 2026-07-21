//go:build !windows

package gofiledialog

func validatePlatformFolderName(string) error {
	return nil
}
