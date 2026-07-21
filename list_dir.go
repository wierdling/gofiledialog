package gofiledialog

import (
	"os"
	"path/filepath"
)

// listDir reads dir and returns its entries as FileEntry values, in
// directory order. Entries that can no longer be stat'd (e.g. removed
// mid-read) are silently skipped. Hidden entries (dotfiles on unix, the
// hidden attribute on Windows) are omitted unless showHidden is true.
// Callers are expected to sort the result (see sortEntries).
func listDir(dir string, showHidden bool, filters ...func(FileEntry) bool) ([]FileEntry, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var filter func(FileEntry) bool
	if len(filters) > 0 {
		filter = filters[0]
	}

	entries := make([]FileEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		name := de.Name()
		linkInfo, err := de.Info()
		if err != nil {
			continue
		}
		info := linkInfo
		if info.Mode()&os.ModeSymlink != 0 {
			info, err = os.Stat(filepath.Join(dir, name))
			if err != nil {
				continue
			}
		}
		entry := newFileEntry(dir, info)
		entry.Name = name
		entry.Path = filepath.Join(dir, name)
		// de.Info follows symlinks above, but visibility belongs to the name
		// shown in this directory, not to the target's name.
		entry.Hidden = isHiddenName(name, linkInfo)
		if entry.Hidden && !showHidden {
			continue
		}
		if filter != nil && !entry.IsDir && !filter(entry) {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
