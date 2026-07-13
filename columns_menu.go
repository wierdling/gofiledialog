package gofiledialog

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// newColumnsMenu builds a "Columns" toolbar button that opens a popup with a
// checkbox per toggleable column (Name is always on, so it's omitted).
// Column visibility only affects the Details view, so callers should disable
// the returned button while another view is active.
func newColumnsMenu(win fyne.Window, b *Browser) *widget.Button {
	btn := widget.NewButton("Columns", nil)
	btn.OnTapped = func() {
		var items []fyne.CanvasObject
		for _, col := range b.Columns() {
			if col.ID == ColName {
				continue
			}
			col := col
			check := widget.NewCheck(col.Title, func(checked bool) {
				b.SetColumnVisible(col.ID, checked)
			})
			check.SetChecked(col.Visible)
			items = append(items, check)
		}

		popup := widget.NewPopUp(container.NewVBox(items...), win.Canvas())
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(btn)
		popup.ShowAtPosition(fyne.NewPos(pos.X, pos.Y+btn.Size().Height))
	}
	return btn
}
