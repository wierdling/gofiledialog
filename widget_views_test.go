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
	check := cell.(*fyne.Container).Objects[0].(*widget.Check)
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
	check := row.(*fyne.Container).Objects[0].(*widget.Check)
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
		obj := cell.(*fyne.Container)
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
	if check := detailsCell.(*fyne.Container).Objects[0].(*widget.Check); !check.Checked {
		t.Fatal("details checkbox did not reflect row click")
	}
	if check := listCell.(*fyne.Container).Objects[0].(*widget.Check); !check.Checked {
		t.Fatal("list checkbox did not reflect row click")
	}
	if check := gridCell.(*fyne.Container).Objects[0].(*widget.Check); !check.Checked {
		t.Fatal("icon checkbox did not reflect row click")
	}
}
