package gofiledialog

import (
	"sort"
	"strings"
)

// sortEntries sorts entries by col in the given direction, always keeping
// directories before files, with a stable alphabetical tie-break by name.
func sortEntries(entries []FileEntry, col Column, asc bool) {
	less := col.Less
	if !asc {
		less = func(a, b FileEntry) bool { return col.Less(b, a) }
	}
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		if less(a, b) {
			return true
		}
		if less(b, a) {
			return false
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
}
