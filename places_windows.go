//go:build windows

package gofiledialog

import (
	"fmt"
	"os"
)

// listPlaces returns the sidebar shortcuts on Windows: the user's known
// folders followed by every mounted drive letter.
func listPlaces() []Place {
	var places []Place
	if home, err := os.UserHomeDir(); err == nil && existingDir(home) {
		places = append(places, homePlaces(home)...)
	}
	for c := 'A'; c <= 'Z'; c++ {
		root := fmt.Sprintf("%c:\\", c)
		if existingDir(root) {
			places = append(places, Place{Name: root, Path: root})
		}
	}
	return places
}
