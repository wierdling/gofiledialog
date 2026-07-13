package gofiledialog

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// fixedWidthLayout gives its (single) child a fixed width while letting it
// fill whatever height the parent layout grants — used to keep the places
// sidebar a constant width regardless of its content.
type fixedWidthLayout struct {
	width float32
}

func (f fixedWidthLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var h float32
	for _, o := range objects {
		if m := o.MinSize().Height; m > h {
			h = m
		}
	}
	return fyne.NewSize(f.width, h)
}

func (f fixedWidthLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Resize(size)
		o.Move(fyne.NewPos(0, 0))
	}
}

const sidebarWidth = 160

// newPlacesSidebar builds the places list wired to navigate b. onError, if
// non-nil, is called when navigating to a place fails.
func newPlacesSidebar(b *Browser, onError func(error)) fyne.CanvasObject {
	places := listPlaces()
	list := widget.NewList(
		func() int { return len(places) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(places[id].Name)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(places) {
			return
		}
		if err := b.NavigateTo(places[id].Path); err != nil && onError != nil {
			onError(err)
		}
	}
	return container.New(fixedWidthLayout{width: sidebarWidth}, list)
}
