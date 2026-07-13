//go:build !windows

package gofiledialog

import "os"

// listPlaces returns the sidebar shortcuts on non-Windows platforms: the
// user's known folders plus the filesystem root. Full mount enumeration and
// XDG user-dirs support are left for a later pass.
func listPlaces() []Place {
	var places []Place
	if home, err := os.UserHomeDir(); err == nil && existingDir(home) {
		places = append(places, homePlaces(home)...)
	}
	places = append(places, Place{Name: "Computer", Path: "/"})
	return places
}
