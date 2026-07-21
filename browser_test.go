package gofiledialog

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func TestBrowserMoveSelectionClampsToListing(t *testing.T) {
	b := testBrowserWithEntries("alpha.txt", "bravo.txt")

	b.MoveSelection(1)
	if b.selectedRow != 0 {
		t.Fatalf("selectedRow = %d, want 0", b.selectedRow)
	}
	b.MoveSelection(99)
	if b.selectedRow != 1 {
		t.Fatalf("selectedRow = %d, want 1", b.selectedRow)
	}
	b.MoveSelection(-99)
	if b.selectedRow != 0 {
		t.Fatalf("selectedRow = %d, want 0", b.selectedRow)
	}
}

func TestBrowserTypeAheadWrapsFromCurrentSelection(t *testing.T) {
	b := testBrowserWithEntries("alpha.txt", "bravo.txt", "beta.txt")
	b.selectedRow = 1

	if !b.TypeAhead("b") {
		t.Fatal("expected type-ahead match")
	}
	if b.selectedRow != 2 {
		t.Fatalf("selectedRow = %d, want 2", b.selectedRow)
	}

	if !b.TypeAhead("a") {
		t.Fatal("expected wrapped type-ahead match")
	}
	if b.selectedRow != 0 {
		t.Fatalf("selectedRow = %d, want 0", b.selectedRow)
	}
}

func TestBrowserCloseClosesThumbnailer(t *testing.T) {
	b, err := NewBrowser(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	b.Close()
	b.Close()

	called := false
	b.thumbs.Request(FileEntry{Name: "image.png", Size: 100}, 32, func(fyne.Resource) {
		called = true
	})
	if called {
		t.Fatal("thumbnail request callback ran after browser close")
	}
}

func TestBrowserSetSortIgnoresUnknownColumnID(t *testing.T) {
	b := testBrowserWithEntries("b.txt", "a.txt")
	b.sortCol = ColName
	b.sortAsc = true
	called := false
	b.OnSettingsChanged = func() { called = true }

	b.SetSort(ColumnID(999))

	if b.sortCol != ColName || !b.sortAsc {
		t.Fatalf("sort = (%v, %v), want unchanged (%v, true)", b.sortCol, b.sortAsc, ColName)
	}
	if called {
		t.Fatal("invalid sort should not trigger settings change")
	}
	if got := names(b.entries); !sameStrings(got, []string{"b.txt", "a.txt"}) {
		t.Fatalf("entries = %v, want unchanged order", got)
	}
}

func testBrowserWithEntries(names ...string) *Browser {
	entries := make([]FileEntry, len(names))
	for i, name := range names {
		entries[i] = FileEntry{Name: name}
	}
	return &Browser{
		entries:      entries,
		selectedRow:  -1,
		selectedRows: make(map[int]bool),
		grids:        make(map[ViewMode]*widget.GridWrap),
		viewMode:     ViewDetails,
	}
}
