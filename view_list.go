package gofiledialog

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// newEntryList builds the compact List view: one row per entry with a small
// file-type icon and the name, no other columns.
func newEntryList(b *Browser) *widget.List {
	paths := make(map[*widget.Check]string)
	suppress := make(map[*widget.Check]bool)
	list := widget.NewList(
		func() int { return len(b.entries) },
		func() fyne.CanvasObject {
			check := widget.NewCheck("", nil)
			check.Hide()
			icon := widget.NewIcon(nil)
			label := widget.NewLabel("")
			paths[check] = ""
			check.OnChanged = func(checked bool) {
				if !suppress[check] {
					b.setPathSelected(paths[check], checked)
				}
			}
			return container.NewHBox(check, icon, label)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			row := obj.(*fyne.Container)
			check := row.Objects[0].(*widget.Check)
			icon := row.Objects[1].(*widget.Icon)
			label := row.Objects[2].(*widget.Label)
			if id < 0 || id >= len(b.entries) {
				paths[check] = ""
				check.Hide()
				label.SetText("")
				return
			}
			entry := b.entries[id]
			if b.multi && isRegularFileEntry(entry) {
				paths[check] = entry.Path
				suppress[check] = true
				check.SetChecked(b.selectedRows[entry.Path])
				suppress[check] = false
				check.Show()
			} else {
				paths[check] = ""
				check.Hide()
			}
			icon.SetResource(entryIconResource(entry))
			label.SetText(entry.Name)
		},
	)
	list.OnSelected = func(id widget.ListItemID) { b.onEntryTapped(id) }
	return list
}
