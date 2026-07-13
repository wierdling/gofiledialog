package gofiledialog

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// newEntryList builds the compact List view: one row per entry with a small
// file-type icon and the name, no other columns.
func newEntryList(b *Browser) *widget.List {
	list := widget.NewList(
		func() int { return len(b.entries) },
		func() fyne.CanvasObject {
			icon := widget.NewIcon(nil)
			label := widget.NewLabel("")
			return container.NewHBox(icon, label)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			row := obj.(*fyne.Container)
			icon := row.Objects[0].(*widget.Icon)
			label := row.Objects[1].(*widget.Label)
			if id < 0 || id >= len(b.entries) {
				label.SetText("")
				return
			}
			entry := b.entries[id]
			icon.SetResource(entryIconResource(entry))
			label.SetText(entry.Name)
		},
	)
	list.OnSelected = func(id widget.ListItemID) { b.onEntryTapped(id) }
	return list
}
