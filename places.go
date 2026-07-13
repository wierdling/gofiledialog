package gofiledialog

import (
	"os"
	"path/filepath"
)

// Place is a shortcut shown in the dialog's sidebar (a drive, a known
// folder, etc).
type Place struct {
	Name string
	Path string
}

func existingDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// homePlaces returns the user's home directory plus any of the standard
// known folders (Desktop, Documents, Downloads, Pictures) that exist under
// it. Shared by the Windows and non-Windows place listings.
func homePlaces(home string) []Place {
	places := []Place{{Name: "Home", Path: home}}
	for _, name := range []string{"Desktop", "Documents", "Downloads", "Pictures"} {
		if p := filepath.Join(home, name); existingDir(p) {
			places = append(places, Place{Name: name, Path: p})
		}
	}
	return places
}
