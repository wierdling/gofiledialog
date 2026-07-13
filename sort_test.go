package gofiledialog

import "testing"

func TestSortEntriesKeepsDirectoriesFirst(t *testing.T) {
	entries := []FileEntry{
		{Name: "zeta.txt", Size: 1},
		{Name: "beta", IsDir: true},
		{Name: "alpha.txt", Size: 3},
		{Name: "alpha", IsDir: true},
	}

	sortEntries(entries, defaultColumns()[0], true)

	got := names(entries)
	want := []string{"alpha", "beta", "alpha.txt", "zeta.txt"}
	if !sameStrings(got, want) {
		t.Fatalf("sorted names = %v, want %v", got, want)
	}
}

func TestSortEntriesUsesNameTieBreak(t *testing.T) {
	entries := []FileEntry{
		{Name: "bravo.txt", Size: 10},
		{Name: "alpha.txt", Size: 10},
	}

	sortEntries(entries, defaultColumns()[3], true)

	got := names(entries)
	want := []string{"alpha.txt", "bravo.txt"}
	if !sameStrings(got, want) {
		t.Fatalf("sorted names = %v, want %v", got, want)
	}
}

func names(entries []FileEntry) []string {
	out := make([]string, len(entries))
	for i, entry := range entries {
		out[i] = entry.Name
	}
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
