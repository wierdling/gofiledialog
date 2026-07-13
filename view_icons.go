package gofiledialog

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// iconViewSize is the icon pixel size and overall grid cell size for one of
// the three icon views.
type iconViewSize struct {
	icon  float32
	cellW float32
	cellH float32
}

var (
	smallIconSize  = iconViewSize{icon: 48, cellW: 90, cellH: 90}
	mediumIconSize = iconViewSize{icon: 96, cellW: 140, cellH: 140}
	largeIconSize  = iconViewSize{icon: 160, cellW: 200, cellH: 200}
)

// fixedSizeLayout gives its (single) child a constant size regardless of its
// natural content size — used to keep every icon-view grid cell uniform.
type fixedSizeLayout struct {
	size fyne.Size
}

func (f fixedSizeLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return f.size
}

func (f fixedSizeLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Resize(size)
		o.Move(fyne.NewPos(0, 0))
	}
}

// newIconGrid builds an icon-view widget.GridWrap at the given size preset.
// Each cell shows the file-type icon immediately, with a "Loading…" overlay
// while a real thumbnail is pending for image files, and swaps in the
// thumbnail (or clears the overlay, keeping the file-type icon) once the
// thumbnailer resolves it.
func newIconGrid(b *Browser, sz iconViewSize) *widget.GridWrap {
	// GridWrap recycles cell widgets as the user scrolls, so an async
	// thumbnail that arrives after its cell has been reused for a different
	// entry must not overwrite that cell. current tracks, per image widget,
	// which entry's thumbnail request is still relevant.
	var mu sync.Mutex
	current := map[*canvas.Image]string{}

	grid := widget.NewGridWrap(
		func() int { return len(b.entries) },
		func() fyne.CanvasObject {
			img := canvas.NewImageFromResource(nil)
			img.FillMode = canvas.ImageFillContain
			img.SetMinSize(fyne.NewSize(sz.icon, sz.icon))

			loading := widget.NewLabel("Loading…")
			loading.Alignment = fyne.TextAlignCenter
			loading.Hide()

			nameLabel := widget.NewLabel("")
			nameLabel.Alignment = fyne.TextAlignCenter
			nameLabel.Wrapping = fyne.TextWrapWord

			imgArea := container.NewStack(container.NewCenter(img), container.NewCenter(loading))
			content := container.NewVBox(imgArea, nameLabel)
			return container.New(fixedSizeLayout{size: fyne.NewSize(sz.cellW, sz.cellH)}, content)
		},
		func(id widget.GridWrapItemID, obj fyne.CanvasObject) {
			cell := obj.(*fyne.Container)
			content := cell.Objects[0].(*fyne.Container)
			imgArea := content.Objects[0].(*fyne.Container)
			img := imgArea.Objects[0].(*fyne.Container).Objects[0].(*canvas.Image)
			loading := imgArea.Objects[1].(*fyne.Container).Objects[0].(*widget.Label)
			nameLabel := content.Objects[1].(*widget.Label)

			if id < 0 || id >= len(b.entries) {
				nameLabel.SetText("")
				loading.Hide()
				return
			}
			entry := b.entries[id]
			nameLabel.SetText(entry.Name)
			img.Resource = entryIconResource(entry)
			img.Refresh()

			mu.Lock()
			current[img] = entry.Path
			mu.Unlock()

			if b.thumbs.MightThumbnail(entry) {
				loading.Show()
			} else {
				loading.Hide()
			}

			b.thumbs.Request(entry, int(sz.icon), func(res fyne.Resource) {
				mu.Lock()
				stale := current[img] != entry.Path
				mu.Unlock()
				if stale {
					return
				}
				loading.Hide()
				if res != nil {
					img.Resource = res
					img.Refresh()
				}
			})
		},
	)
	grid.OnSelected = func(id widget.GridWrapItemID) { b.onEntryTapped(id) }
	return grid
}
