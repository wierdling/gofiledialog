package gofiledialog

import (
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const doubleClickThreshold = 400 * time.Millisecond

// entryView is the subset of widget.Table/widget.List/widget.GridWrap that
// Browser needs to reset/refresh uniformly regardless of which view is
// currently active.
type entryView interface {
	fyne.CanvasObject
	UnselectAll()
	ScrollToTop()
}

// Browser is the core directory-browsing widget shared by the Open, Save and
// Folder dialogs. It supports Details, List, and Small/Medium/Large icon
// views over the same underlying entries.
type Browser struct {
	dir        string
	entries    []FileEntry
	showHidden bool
	filter     func(FileEntry) bool
	multi      bool

	table *widget.Table
	list  *widget.List
	grids map[ViewMode]*widget.GridWrap
	views []entryView

	viewMode ViewMode
	stack    *fyne.Container
	thumbs   *thumbnailer

	columns []Column
	sortCol ColumnID
	sortAsc bool

	history   []string
	histIndex int

	selectedRow   int
	selectedRows  map[int]bool
	selectingRow  bool
	lastClickRow  int
	lastClickTime time.Time

	// OnActivate fires when the user activates a file row. Directories are
	// navigated into as soon as they are selected because Fyne's collection
	// widgets do not emit a second OnSelected event for the already-selected
	// item.
	OnActivate func(entry FileEntry)

	// OnDirChanged fires whenever the browser finishes navigating to a new
	// directory, so a toolbar/breadcrumb can refresh its state.
	OnDirChanged func(dir string)

	// OnSelectionChanged fires when the selected entry set changes.
	OnSelectionChanged func()

	// OnError fires when a navigation attempt (double-click, Back/Forward/Up,
	// or a path typed into a breadcrumb) fails, e.g. permission denied.
	OnError func(err error)

	// OnSettingsChanged fires whenever persisted dialog state (sort, column
	// visibility/width, show-hidden) changes, so a caller can save it.
	OnSettingsChanged func()
}

// NewBrowser creates a Browser listing startDir.
func NewBrowser(startDir string) (*Browser, error) {
	b := &Browser{
		selectedRow:  -1,
		selectedRows: make(map[int]bool),
		lastClickRow: -1,
		histIndex:    -1,
		columns:      defaultColumns(),
		sortCol:      ColName,
		sortAsc:      true,
		viewMode:     ViewDetails,
		thumbs:       newThumbnailer(),
	}
	b.table = newDetailsTable(b)
	b.list = newEntryList(b)
	b.grids = map[ViewMode]*widget.GridWrap{
		ViewSmallIcons:  newIconGrid(b, smallIconSize),
		ViewMediumIcons: newIconGrid(b, mediumIconSize),
		ViewLargeIcons:  newIconGrid(b, largeIconSize),
	}
	b.views = []entryView{b.table, b.list, b.grids[ViewSmallIcons], b.grids[ViewMediumIcons], b.grids[ViewLargeIcons]}

	b.stack = container.NewStack(b.table, b.list,
		b.grids[ViewSmallIcons], b.grids[ViewMediumIcons], b.grids[ViewLargeIcons])
	b.showActiveView()

	if err := b.NavigateTo(startDir); err != nil {
		return nil, err
	}
	return b, nil
}

// VisibleColumns returns the currently visible columns in display order.
func (b *Browser) VisibleColumns() []Column {
	visible := make([]Column, 0, len(b.columns))
	for _, col := range b.columns {
		if col.Visible {
			visible = append(visible, col)
		}
	}
	return visible
}

// Columns returns every registered column (visible or not), for building a
// "choose columns" menu.
func (b *Browser) Columns() []Column {
	return append([]Column(nil), b.columns...)
}

// SetColumnVisible shows or hides a column. Name cannot be hidden.
func (b *Browser) SetColumnVisible(id ColumnID, visible bool) {
	if id == ColName {
		return
	}
	for i := range b.columns {
		if b.columns[i].ID == id {
			b.columns[i].Visible = visible
			break
		}
	}
	b.refreshColumnWidths()
	b.table.Refresh()
	b.settingsChanged()
}

// SetFilter sets a display filter for files and refreshes the current
// directory. Directories are always shown so users can keep navigating.
func (b *Browser) SetFilter(filter func(FileEntry) bool) error {
	b.filter = filter
	entries, err := listDir(b.dir, b.showHidden, b.filter)
	if err != nil {
		return err
	}
	b.applyListing(b.dir, entries)
	return nil
}

// SetMultiSelect controls whether selected files accumulate. Directory
// activation still navigates immediately.
func (b *Browser) SetMultiSelect(enabled bool) {
	b.multi = enabled
	b.clearSelection()
}

// SortColumn and SortAscending report the current Details-view sort state.
func (b *Browser) SortColumn() ColumnID { return b.sortCol }
func (b *Browser) SortAscending() bool  { return b.sortAsc }

// SetSort sorts the Details view by column id, toggling direction if id is
// already the active sort column.
func (b *Browser) SetSort(id ColumnID) {
	if b.sortCol == id {
		b.sortAsc = !b.sortAsc
	} else {
		b.sortCol = id
		b.sortAsc = true
	}
	sortEntries(b.entries, b.columnByID(id), b.sortAsc)
	b.table.Refresh()
	b.settingsChanged()
}

func (b *Browser) settingsChanged() {
	if b.OnSettingsChanged != nil {
		b.OnSettingsChanged()
	}
}

func (b *Browser) columnByID(id ColumnID) Column {
	for _, col := range b.columns {
		if col.ID == id {
			return col
		}
	}
	return b.columns[0]
}

// applySettings applies loaded persisted state (column visibility/widths,
// sort, show-hidden) to a freshly constructed Browser, then re-lists the
// current directory so the changes take effect. It must be called before
// the dialog is shown, and does not itself trigger OnSettingsChanged.
func (b *Browser) applySettings(s Settings) {
	s = normalizeSettings(s)
	b.columns = columnsFromSettings(s.Columns)
	b.sortCol = s.SortColumn
	b.sortAsc = s.SortAscending
	b.showHidden = s.ShowHidden
	b.viewMode = viewModeFromString(s.ViewMode)
	b.showActiveView()

	if entries, err := listDir(b.dir, b.showHidden, b.filter); err == nil {
		b.applyListing(b.dir, entries)
	}
	b.refreshColumnWidths()
	b.table.Refresh()
}

func columnsFromSettings(settings []ColumnSetting) []Column {
	defaults := defaultColumns()
	byID := make(map[ColumnID]Column, len(defaults))
	for _, col := range defaults {
		byID[col.ID] = col
	}

	normalized := normalizeColumnSettings(settings)
	cols := make([]Column, 0, len(normalized))
	for _, cs := range normalized {
		col, ok := byID[cs.ID]
		if !ok {
			continue
		}
		col.Visible = cs.Visible
		if col.ID == ColName {
			col.Visible = true
		}
		col.Width = cs.Width
		cols = append(cols, col)
	}
	if len(cols) == 0 {
		return defaults
	}
	return cols
}

// NavigateTo lists dir, replaces the browser's current contents, and pushes
// dir onto the navigation history (discarding any forward history).
func (b *Browser) NavigateTo(dir string) error {
	dir = filepath.Clean(dir)
	entries, err := listDir(dir, b.showHidden, b.filter)
	if err != nil {
		return err
	}
	if b.histIndex >= 0 {
		b.history = b.history[:b.histIndex+1]
	}
	b.history = append(b.history, dir)
	b.histIndex = len(b.history) - 1
	b.applyListing(dir, entries)
	return nil
}

// Up navigates to the parent of the current directory, if any.
func (b *Browser) Up() error {
	parent := filepath.Dir(b.dir)
	if parent == b.dir {
		return nil
	}
	return b.NavigateTo(parent)
}

// Back navigates to the previous directory in history, if any.
func (b *Browser) Back() error {
	if !b.CanGoBack() {
		return nil
	}
	return b.goToHistoryIndex(b.histIndex - 1)
}

// Forward navigates to the next directory in history, if any.
func (b *Browser) Forward() error {
	if !b.CanGoForward() {
		return nil
	}
	return b.goToHistoryIndex(b.histIndex + 1)
}

// CanGoBack reports whether Back would move to a different directory.
func (b *Browser) CanGoBack() bool {
	return b.histIndex > 0
}

// CanGoForward reports whether Forward would move to a different directory.
func (b *Browser) CanGoForward() bool {
	return b.histIndex < len(b.history)-1
}

// ToggleShowHidden sets whether hidden files/directories are listed and
// re-lists the current directory.
func (b *Browser) ToggleShowHidden(show bool) error {
	b.showHidden = show
	entries, err := listDir(b.dir, b.showHidden, b.filter)
	if err != nil {
		return err
	}
	b.applyListing(b.dir, entries)
	b.settingsChanged()
	return nil
}

// ShowHidden reports whether hidden files/directories are currently listed.
func (b *Browser) ShowHidden() bool {
	return b.showHidden
}

func (b *Browser) goToHistoryIndex(idx int) error {
	dir := b.history[idx]
	entries, err := listDir(dir, b.showHidden, b.filter)
	if err != nil {
		return err
	}
	b.histIndex = idx
	b.applyListing(dir, entries)
	return nil
}

// applyListing updates the browser's displayed contents. It does not touch
// navigation history; callers are responsible for that.
func (b *Browser) applyListing(dir string, entries []FileEntry) {
	// Do not let thumbnails for the previous directory occupy the queue while
	// the newly visible cells are waiting to be populated.
	b.thumbs.CancelPending()
	sortEntries(entries, b.columnByID(b.sortCol), b.sortAsc)
	b.dir = dir
	b.entries = entries
	b.selectedRow = -1
	b.selectedRows = make(map[int]bool)
	b.lastClickRow = -1
	for _, v := range b.views {
		v.UnselectAll()
		v.ScrollToTop()
		v.Refresh()
	}
	if b.OnDirChanged != nil {
		b.OnDirChanged(dir)
	}
	b.selectionChanged()
}

// CurrentDir returns the directory currently being browsed.
func (b *Browser) CurrentDir() string {
	return b.dir
}

// Selected returns the currently highlighted entry, if any.
func (b *Browser) Selected() (FileEntry, bool) {
	if b.selectedRow < 0 || b.selectedRow >= len(b.entries) {
		return FileEntry{}, false
	}
	return b.entries[b.selectedRow], true
}

// SelectedEntries returns selected files in display order. Directories are
// intentionally excluded for Open dialogs; Folder dialogs use CurrentDir.
func (b *Browser) SelectedEntries() []FileEntry {
	if len(b.selectedRows) == 0 {
		if entry, ok := b.Selected(); ok && !entry.IsDir {
			return []FileEntry{entry}
		}
		return nil
	}
	entries := make([]FileEntry, 0, len(b.selectedRows))
	for i, entry := range b.entries {
		if b.selectedRows[i] && !entry.IsDir {
			entries = append(entries, entry)
		}
	}
	return entries
}

// MoveSelection moves the current row highlight by delta, clamped to the
// current listing. It is used by keyboard navigation.
func (b *Browser) MoveSelection(delta int) {
	if len(b.entries) == 0 {
		return
	}
	index := b.selectedRow
	if index < 0 {
		if delta < 0 {
			index = len(b.entries) - 1
		} else {
			index = 0
		}
	} else {
		index += delta
	}
	if index < 0 {
		index = 0
	}
	if index >= len(b.entries) {
		index = len(b.entries) - 1
	}
	b.selectRow(index)
}

// ActivateSelected opens the selected directory or activates the selected
// file through OnActivate.
func (b *Browser) ActivateSelected() {
	entry, ok := b.Selected()
	if !ok {
		return
	}
	if entry.IsDir {
		if err := b.NavigateTo(entry.Path); err != nil && b.OnError != nil {
			b.OnError(err)
		}
		return
	}
	if b.OnActivate != nil {
		b.OnActivate(entry)
	}
}

// TypeAhead selects the next entry whose name begins with prefix, wrapping
// around from the current selection.
func (b *Browser) TypeAhead(prefix string) bool {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" || len(b.entries) == 0 {
		return false
	}
	start := b.selectedRow + 1
	if start < 0 || start >= len(b.entries) {
		start = 0
	}
	for offset := 0; offset < len(b.entries); offset++ {
		index := (start + offset) % len(b.entries)
		if strings.HasPrefix(strings.ToLower(b.entries[index].Name), prefix) {
			b.selectRow(index)
			return true
		}
	}
	return false
}

func (b *Browser) Refresh() error {
	entries, err := listDir(b.dir, b.showHidden, b.filter)
	if err != nil {
		return err
	}
	b.applyListing(b.dir, entries)
	return nil
}

// Content returns the canvas object to embed in a dialog window.
func (b *Browser) Content() fyne.CanvasObject {
	return b.stack
}

// onEntryTapped handles a single-click/tap on entries[index] from whichever
// view widget is currently active (Table, List, or GridWrap all funnel
// through here), tracking selection and detecting double-clicks.
func (b *Browser) onEntryTapped(index int) {
	if index < 0 || index >= len(b.entries) {
		return
	}
	if b.selectingRow {
		return
	}
	b.selectedRow = index

	entry := b.entries[index]
	if entry.IsDir {
		if err := b.NavigateTo(entry.Path); err != nil && b.OnError != nil {
			b.OnError(err)
		}
		return
	}
	if b.multi {
		b.selectedRows[index] = true
	} else {
		b.selectedRows = map[int]bool{index: true}
	}
	b.selectionChanged()

	// Keep file activation separate from selection so the Open button remains
	// the explicit action for files. The click timing fields are retained for
	// compatibility with the existing browser state and future activation
	// handling.
	now := time.Now()
	isDoubleClick := index == b.lastClickRow && now.Sub(b.lastClickTime) < doubleClickThreshold
	b.lastClickRow = index
	b.lastClickTime = now
	if !isDoubleClick {
		return
	}
	if b.OnActivate != nil {
		b.OnActivate(entry)
	}
}

func (b *Browser) selectRow(index int) {
	if index < 0 || index >= len(b.entries) {
		return
	}
	b.selectedRow = index
	if !b.multi && !b.entries[index].IsDir {
		b.selectedRows = map[int]bool{index: true}
	}
	b.selectingRow = true
	switch b.viewMode {
	case ViewDetails:
		if b.table != nil {
			b.table.Select(widget.TableCellID{Row: index, Col: 0})
		}
	case ViewList:
		if b.list != nil {
			b.list.Select(index)
		}
	default:
		if grid, ok := b.grids[b.viewMode]; ok && grid != nil {
			grid.Select(index)
		}
	}
	b.selectingRow = false
	b.selectionChanged()
}

func (b *Browser) clearSelection() {
	b.selectedRow = -1
	b.selectedRows = make(map[int]bool)
	for _, v := range b.views {
		v.UnselectAll()
	}
	b.selectionChanged()
}

func (b *Browser) selectionChanged() {
	if b.OnSelectionChanged != nil {
		b.OnSelectionChanged()
	}
}
