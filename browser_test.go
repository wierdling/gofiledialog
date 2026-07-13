package gofiledialog

import (
	"testing"

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
