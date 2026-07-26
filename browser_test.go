package gofiledialog

import (
	"os"
	"path/filepath"
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

func TestBrowserMultiSelectionIsExplicitAndPathKeyed(t *testing.T) {
	b := testBrowserWithEntries("b.txt", "a.txt", "dir")
	b.multi = true
	b.entries[0].Path = filepath.Join("/tmp", "b.txt")
	b.entries[1].Path = filepath.Join("/tmp", "a.txt")
	b.entries[2].Path = filepath.Join("/tmp", "dir")
	b.entries[2].IsDir = true
	b.entries[0].Mode = 0
	b.entries[1].Mode = 0

	// Merely moving the active row does not implicitly select it in multi mode.
	b.selectedRow = 1
	if got := b.SelectedEntries(); len(got) != 0 {
		t.Fatalf("SelectedEntries before explicit click = %v, want empty", got)
	}
	b.onEntryTapped(0)
	b.onEntryTapped(1)
	if got := b.SelectedEntries(); len(got) != 2 || got[0].Path != b.entries[0].Path || got[1].Path != b.entries[1].Path {
		t.Fatalf("SelectedEntries = %v, want selected files in display order", got)
	}

	// Sorting changes display indexes but must retain membership by path.
	sortEntries(b.entries, b.columnByID(ColName), false)
	got := b.SelectedEntries()
	if len(got) != 2 || got[0].Path != filepath.Join("/tmp", "b.txt") || got[1].Path != filepath.Join("/tmp", "a.txt") {
		t.Fatalf("SelectedEntries after sort = %v, want selected files in descending display order", got)
	}
}

func TestBrowserListingClearsSelection(t *testing.T) {
	b := testBrowserWithEntries("a.txt")
	b.thumbs = newThumbnailer()
	defer b.Close()
	b.multi = true
	b.entries[0].Path = filepath.Join("/tmp", "a.txt")
	b.onEntryTapped(0)
	if len(b.SelectedEntries()) != 1 {
		t.Fatal("expected explicit selection")
	}
	b.applyListing("/tmp", []FileEntry{{Name: "a.txt", Path: b.entries[0].Path}})
	if got := b.SelectedEntries(); len(got) != 0 {
		t.Fatalf("SelectedEntries after listing refresh = %v, want empty", got)
	}
}

func TestBrowserSelectedEntriesExcludesSpecialFiles(t *testing.T) {
	b := testBrowserWithEntries("pipe")
	b.entries[0].Path = filepath.Join("/tmp", "pipe")
	b.entries[0].Mode = 0o644 | 0 // regular synthetic entry
	b.entries = append(b.entries, FileEntry{Name: "socket", Path: filepath.Join("/tmp", "socket"), Mode: os.ModeNamedPipe})
	b.multi = true
	b.onEntryTapped(0)
	b.onEntryTapped(1)
	got := b.SelectedEntries()
	if len(got) != 1 || got[0].Name != "pipe" {
		t.Fatalf("SelectedEntries = %v, want only regular files", got)
	}
}

func TestBrowserCheckboxSelectionIsPathKeyed(t *testing.T) {
	b := testBrowserWithEntries("one.txt", "two.txt", "folder")
	b.multi = true
	b.entries[0].Path = filepath.Join("/tmp", "one.txt")
	b.entries[1].Path = filepath.Join("/tmp", "two.txt")
	b.entries[2].Path = filepath.Join("/tmp", "folder")
	b.entries[2].IsDir = true

	b.setPathSelected(b.entries[1].Path, true)
	if got := b.SelectedEntries(); len(got) != 1 || got[0].Path != b.entries[1].Path {
		t.Fatalf("selected entries = %v, want two.txt", got)
	}
	// A recycled checkbox for a directory or an unknown path must not mutate
	// the explicit selection map.
	b.setPathSelected(b.entries[2].Path, true)
	b.setPathSelected(filepath.Join("/tmp", "missing.txt"), true)
	if len(b.selectedRows) != 1 {
		t.Fatalf("selectedRows = %v, want one path", b.selectedRows)
	}
	b.setPathSelected(b.entries[1].Path, false)
	if got := b.SelectedEntries(); len(got) != 0 {
		t.Fatalf("selected entries after uncheck = %v, want empty", got)
	}
}

func TestBrowserToggleFocusedSelectionOnlyInMultiMode(t *testing.T) {
	b := testBrowserWithEntries("one.txt", "pipe")
	b.entries[0].Path = filepath.Join("/tmp", "one.txt")
	b.entries[0].Mode = 0o644
	b.entries[1].Path = filepath.Join("/tmp", "pipe")
	b.entries[1].Mode = os.ModeNamedPipe
	b.multi = true

	b.selectRow(0)
	if !b.ToggleFocusedSelection() || len(b.SelectedEntries()) != 1 {
		t.Fatalf("toggle did not check focused regular file: %v", b.SelectedEntries())
	}
	if !b.ToggleFocusedSelection() || len(b.SelectedEntries()) != 0 {
		t.Fatalf("second toggle did not uncheck focused regular file: %v", b.SelectedEntries())
	}
	b.selectRow(1)
	if b.ToggleFocusedSelection() {
		t.Fatal("toggle reported success for focused special file")
	}
}

func TestBrowserSpecialClickClearsSingleSelectionAndNotifies(t *testing.T) {
	b := testBrowserWithEntries("regular.txt", "pipe")
	b.entries[0].Path = filepath.Join("/tmp", "regular.txt")
	b.entries[0].Mode = 0o644
	b.entries[1].Path = filepath.Join("/tmp", "pipe")
	b.entries[1].Mode = os.ModeNamedPipe

	notifications := 0
	b.OnSelectionChanged = func() { notifications++ }
	b.onEntryTapped(0)
	if got := b.SelectedEntries(); len(got) != 1 || got[0].Name != "regular.txt" {
		t.Fatalf("SelectedEntries after regular click = %v, want regular file", got)
	}

	b.onEntryTapped(1)
	if got := b.SelectedEntries(); len(got) != 0 {
		t.Fatalf("SelectedEntries after special click = %v, want empty", got)
	}
	if b.selectedRow != -1 {
		t.Fatalf("selectedRow after special click = %d, want -1", b.selectedRow)
	}
	if notifications != 2 {
		t.Fatalf("selection notifications = %d, want 2", notifications)
	}
}

func TestBrowserKeyboardSelectionExcludesSpecialFiles(t *testing.T) {
	b := testBrowserWithEntries("regular.txt", "pipe", "later.txt")
	b.entries[0].Path = filepath.Join("/tmp", "regular.txt")
	b.entries[0].Mode = 0o644
	b.entries[1].Path = filepath.Join("/tmp", "pipe")
	b.entries[1].Mode = os.ModeNamedPipe
	b.entries[2].Path = filepath.Join("/tmp", "later.txt")
	b.entries[2].Mode = 0o644

	b.selectRow(0)
	if got := b.SelectedEntries(); len(got) != 1 || got[0].Name != "regular.txt" {
		t.Fatalf("SelectedEntries after regular keyboard selection = %v, want regular file", got)
	}
	b.selectRow(1)
	if got := b.SelectedEntries(); len(got) != 0 {
		t.Fatalf("SelectedEntries after special keyboard selection = %v, want empty", got)
	}
	if b.selectedRow != 1 || len(b.selectedRows) != 0 {
		t.Fatalf("special keyboard selection left logical selection: row=%d paths=%v", b.selectedRow, b.selectedRows)
	}

	// Navigation must advance past the highlighted special row to the next
	// regular file, which then becomes the sole logical selection.
	b.MoveSelection(1)
	if b.selectedRow != 2 {
		t.Fatalf("selection after special row = %d, want 2", b.selectedRow)
	}
	if got := b.SelectedEntries(); len(got) != 1 || got[0].Name != "later.txt" {
		t.Fatalf("SelectedEntries after advancing past special row = %v, want later.txt", got)
	}

	// In multi-select mode, an explicit selection must survive keyboard
	// navigation over a special entry, while the special entry itself is never
	// added to the checked selection set.
	b.multi = true
	b.onEntryTapped(2)
	b.selectRow(1)
	if len(b.selectedRows) != 1 || !b.selectedRows[b.entries[2].Path] {
		t.Fatalf("multi-select special keyboard selection changed paths: %v", b.selectedRows)
	}
	if b.selectedRows[b.entries[1].Path] {
		t.Fatalf("multi-select special keyboard selection added path: %v", b.selectedRows)
	}
	if got := b.SelectedEntries(); len(got) != 1 || got[0].Name != "later.txt" {
		t.Fatalf("SelectedEntries after multi-select special row = %v, want later.txt", got)
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
		selectedRows: make(map[string]bool),
		grids:        make(map[ViewMode]*widget.GridWrap),
		viewMode:     ViewDetails,
		columns:      defaultColumns(),
		sortCol:      ColName,
		sortAsc:      true,
	}
}
