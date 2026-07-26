package gofiledialog

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// newDetailsTable builds the Details-view widget.Table backed by b's current
// entries and column configuration. b.VisibleColumns() drives both the
// column count/order and the sortable header buttons.
func newDetailsTable(b *Browser) *widget.Table {
	paths := make(map[*widget.Check]string)
	suppress := make(map[*widget.Check]bool)
	table := widget.NewTable(
		func() (int, int) {
			return len(b.entries), len(b.VisibleColumns())
		},
		func() fyne.CanvasObject {
			check := widget.NewCheck("", nil)
			check.Hide()
			label := widget.NewLabel("")
			paths[check] = ""
			check.OnChanged = func(checked bool) {
				if !suppress[check] {
					b.setPathSelected(paths[check], checked)
				}
			}
			return container.NewHBox(check, label)
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			row := obj.(*fyne.Container)
			check := row.Objects[0].(*widget.Check)
			label := row.Objects[1].(*widget.Label)
			cols := b.VisibleColumns()
			if id.Row < 0 || id.Row >= len(b.entries) || id.Col >= len(cols) {
				paths[check] = ""
				check.Hide()
				label.SetText("")
				return
			}
			entry := b.entries[id.Row]
			col := cols[id.Col]
			label.SetText(col.Text(entry))
			if col.ID == ColName && b.multi && isRegularFileEntry(entry) {
				paths[check] = entry.Path
				suppress[check] = true
				check.SetChecked(b.selectedRows[entry.Path])
				suppress[check] = false
				check.Show()
			} else {
				paths[check] = ""
				check.Hide()
			}
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
