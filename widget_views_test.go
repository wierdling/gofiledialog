package gofiledialog

import (
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func widgetTestBrowser(t *testing.T) *Browser {
	t.Helper()
	b := testBrowserWithEntries("one.txt", "folder", "pipe")
	root := t.TempDir()
	b.entries[0].Path = filepath.Join(root, "one.txt")
	b.entries[0].Mode = 0o644
	b.entries[1].Path = filepath.Join(root, "folder")
	b.entries[1].IsDir = true
	b.entries[2].Path = filepath.Join(root, "pipe")
	b.entries[2].Mode = os.ModeNamedPipe
	b.multi = true
	b.thumbs = newThumbnailer()
	t.Cleanup(b.thumbs.Close)
	return b
}

func TestDetailsCheckboxWidgetBindingAndRecycle(t *testing.T) {
	b := widgetTestBrowser(t)
	changes := 0
	b.OnSelectionChanged = func() { changes++ }
	table := newDetailsTable(b)
	cell := table.CreateCell()
	// Name is the first visible column and owns the checkbox.
	table.UpdateCell(widget.TableCellID{Row: 0, Col: 0}, cell)
	check := cell.(*tappableContainer).content.(*fyne.Container).Objects[0].(*widget.Check)
	if !check.Visible() || check.Checked {
		t.Fatalf("regular file checkbox visibility=%v checked=%v", check.Visible(), check.Checked)
	}
	check.SetChecked(true)
	if !b.selectedRows[b.entries[0].Path] || changes != 1 {
		t.Fatalf("checkbox selection=%v callback count=%d", b.selectedRows, changes)
	}
	// Rebinding the recycled cell must suppress SetChecked's callback and clear
	// the control for a directory.
	table.UpdateCell(widget.TableCellID{Row: 1, Col: 0}, cell)
	if check.Visible() || changes != 1 {
		t.Fatalf("directory recycle visible=%v callback count=%d", check.Visible(), changes)
	}
	table.UpdateCell(widget.TableCellID{Row: 2, Col: 0}, cell)
	if check.Visible() {
		t.Fatal("special file checkbox should be hidden")
	}
}

func TestListCheckboxWidgetBindingAndRecycle(t *testing.T) {
	b := widgetTestBrowser(t)
	list := newEntryList(b)
	row := list.CreateItem()
	list.UpdateItem(0, row)
	check := row.(*tappableContainer).content.(*fyne.Container).Objects[0].(*widget.Check)
	if !check.Visible() {
		t.Fatal("regular file checkbox should be visible")
	}
	check.SetChecked(true)
	if !b.selectedRows[b.entries[0].Path] {
		t.Fatal("checkbox interaction did not select path")
	}
	list.UpdateItem(2, row)
	if check.Visible() {
		t.Fatal("special file checkbox should be hidden")
	}
}

func TestIconCheckboxLayoutAndCrossViewState(t *testing.T) {
	b := widgetTestBrowser(t)
	b.selectedRows[b.entries[0].Path] = true
	for _, preset := range []iconViewSize{smallIconSize, mediumIconSize, largeIconSize} {
		grid := newIconGrid(b, preset)
		cell := grid.CreateItem()
		grid.UpdateItem(0, cell)
		obj := cell.(*tappableContainer).content.(*fyne.Container)
		obj.Resize(fyne.NewSize(preset.cellW, preset.cellH))
		check := obj.Objects[0].(*widget.Check)
		if !check.Visible() || !check.Checked {
			t.Fatalf("preset %+v checkbox visible=%v checked=%v", preset, check.Visible(), check.Checked)
		}
		for _, child := range obj.Objects {
			pos, size := child.Position(), child.Size()
			if pos.X < 0 || pos.Y < 0 || pos.X+size.Width > preset.cellW || pos.Y+size.Height > preset.cellH {
				t.Fatalf("preset %+v child bounds pos=%v size=%v", preset, pos, size)
			}
		}
		grid.UpdateItem(1, cell)
		if check.Visible() {
			t.Fatalf("directory checkbox should be hidden for preset %+v", preset)
		}
		grid.UpdateItem(2, cell)
		if check.Visible() {
			t.Fatalf("special file checkbox should be hidden for preset %+v", preset)
		}
	}
}

func TestMultiRowClickRefreshesCheckboxStateAcrossViews(t *testing.T) {
	b := widgetTestBrowser(t)
	table := newDetailsTable(b)
	list := newEntryList(b)
	grid := newIconGrid(b, smallIconSize)
	b.views = []entryView{table, list, grid}

	detailsCell := table.CreateCell()
	listCell := list.CreateItem()
	gridCell := grid.CreateItem()
	table.UpdateCell(widget.TableCellID{Row: 0, Col: 0}, detailsCell)
	list.UpdateItem(0, listCell)
	grid.UpdateItem(0, gridCell)
	b.onEntryTapped(0)

	// Refresh triggered by the row click must rebind all recycled controls.
	table.UpdateCell(widget.TableCellID{Row: 0, Col: 0}, detailsCell)
	list.UpdateItem(0, listCell)
	grid.UpdateItem(0, gridCell)
	if check := detailsCell.(*tappableContainer).content.(*fyne.Container).Objects[0].(*widget.Check); !check.Checked {
		t.Fatal("details checkbox did not reflect row click")
	}
	if check := listCell.(*tappableContainer).content.(*fyne.Container).Objects[0].(*widget.Check); !check.Checked {
		t.Fatal("list checkbox did not reflect row click")
	}
	if check := gridCell.(*tappableContainer).content.(*fyne.Container).Objects[0].(*widget.Check); !check.Checked {
		t.Fatal("icon checkbox did not reflect row click")
	}
}

func TestEntryCellsRouteEveryTapAndActivateOnSecondTap(t *testing.T) {
	views := []struct {
		name string
		bind func(*Browser) (fyne.CanvasObject, func(fyne.CanvasObject))
	}{
		{name: "details", bind: func(b *Browser) (fyne.CanvasObject, func(fyne.CanvasObject)) {
			table := newDetailsTable(b)
			return table, func(obj fyne.CanvasObject) { table.UpdateCell(widget.TableCellID{Row: 0, Col: 0}, obj) }
		}},
		{name: "list", bind: func(b *Browser) (fyne.CanvasObject, func(fyne.CanvasObject)) {
			list := newEntryList(b)
			return list, func(obj fyne.CanvasObject) { list.UpdateItem(0, obj) }
		}},
		{name: "icons", bind: func(b *Browser) (fyne.CanvasObject, func(fyne.CanvasObject)) {
			grid := newIconGrid(b, smallIconSize)
			return grid, func(obj fyne.CanvasObject) { grid.UpdateItem(0, obj) }
		}},
	}

	for _, view := range views {
		t.Run(view.name, func(t *testing.T) {
			b := widgetTestBrowser(t)
			activated := 0
			b.OnActivate = func(FileEntry) { activated++ }
			obj, update := view.bind(b)

			var item fyne.CanvasObject
			switch collection := obj.(type) {
			case *widget.Table:
				item = collection.CreateCell()
			case *widget.List:
				item = collection.CreateItem()
			case *widget.GridWrap:
				item = collection.CreateItem()
			}
			update(item)
			tappable := item.(*tappableContainer)
			tappable.Tapped(nil)
			if b.selectedRow != 0 || !b.selectedRows[b.entries[0].Path] || activated != 0 {
				t.Fatalf("first tap selectedRow=%d selected=%v activations=%d", b.selectedRow, b.selectedRows, activated)
			}
			tappable.Tapped(nil)
			if activated != 1 {
				t.Fatalf("second tap activations=%d, want 1", activated)
			}
		})
	}
}

func TestDetailsCellTapPreservesClickedColumnFocus(t *testing.T) {
	b := widgetTestBrowser(t)
	table := newDetailsTable(b)
	cell := table.CreateCell()
	table.UpdateCell(widget.TableCellID{Row: 0, Col: 1}, cell)
	var selected widget.TableCellID
	table.OnSelected = func(id widget.TableCellID) { selected = id }
	tappable := cell.(*tappableContainer)
	tappable.Tapped(nil)

	// A second tap in the same cell must still be treated as the same row for
	// activation, while the collection retains the exact clicked cell focus.
	tappable.Tapped(nil)
	if selected != (widget.TableCellID{Row: 0, Col: 1}) {
		t.Fatalf("selected cell=%v, want row 0 column 1", selected)
	}
}
