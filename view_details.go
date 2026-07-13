package gofiledialog

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// newDetailsTable builds the Details-view widget.Table backed by b's current
// entries and column configuration. b.VisibleColumns() drives both the
// column count/order and the sortable header buttons.
func newDetailsTable(b *Browser) *widget.Table {
	table := widget.NewTable(
		func() (int, int) {
			return len(b.entries), len(b.VisibleColumns())
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			cols := b.VisibleColumns()
			if id.Row < 0 || id.Row >= len(b.entries) || id.Col >= len(cols) {
				label.SetText("")
				return
			}
			label.SetText(cols[id.Col].Text(b.entries[id.Row]))
		},
	)
	table.ShowHeaderRow = true
	table.CreateHeader = func() fyne.CanvasObject {
		btn := widget.NewButton("", nil)
		btn.Alignment = widget.ButtonAlignLeading
		return btn
	}
	table.UpdateHeader = func(id widget.TableCellID, obj fyne.CanvasObject) {
		cols := b.VisibleColumns()
		if id.Col >= len(cols) {
			return
		}
		col := cols[id.Col]
		btn := obj.(*widget.Button)
		btn.SetText(headerLabel(col, b.SortColumn(), b.SortAscending()))
		btn.OnTapped = func() {
			b.SetSort(col.ID)
		}
	}
	table.OnSelected = func(id widget.TableCellID) { b.onEntryTapped(id.Row) }
	b.applyColumnWidths(table)
	return table
}

func headerLabel(col Column, sortCol ColumnID, asc bool) string {
	if col.ID != sortCol {
		return col.Title
	}
	if asc {
		return col.Title + " ▲"
	}
	return col.Title + " ▼"
}

// refreshColumnWidths reapplies each visible column's configured width to
// its current table column index. Must be called whenever the set of
// visible columns changes, since indices shift.
func (b *Browser) refreshColumnWidths() {
	b.applyColumnWidths(b.table)
}

func (b *Browser) applyColumnWidths(table *widget.Table) {
	for i, col := range b.VisibleColumns() {
		table.SetColumnWidth(i, col.Width)
	}
}
