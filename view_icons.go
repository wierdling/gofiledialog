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

// iconCellLayout keeps the icon and caption in a fixed grid cell while
// overlaying the selection checkbox in the top-left corner. The checkbox is
// deliberately not part of the vertical content flow, so showing it never
// changes the icon/caption bounds or causes the cell to exceed its preset.
type iconCellLayout struct {
	size fyne.Size
}

func (l iconCellLayout) MinSize([]fyne.CanvasObject) fyne.Size { return l.size }

func (l iconCellLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 3 {
		return
	}
	cellSize := fyne.NewSize(minFloat(size.Width, l.size.Width), minFloat(size.Height, l.size.Height))
	iconH := cellSize.Height - 28
	if iconH < 0 {
		iconH = 0
	}
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(fyne.NewSize(24, 24))
	objects[1].Move(fyne.NewPos(0, 0))
	objects[1].Resize(fyne.NewSize(cellSize.Width, iconH))
	labelW := cellSize.Width - 4
	if labelW < 0 {
		labelW = 0
	}
	objects[2].Move(fyne.NewPos(2, cellSize.Height-26))
	objects[2].Resize(fyne.NewSize(labelW, 24))
}

func minFloat(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
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
	checks := map[*widget.Check]string{}
	suppress := map[*widget.Check]bool{}

	grid := widget.NewGridWrap(
		func() int { return len(b.entries) },
		func() fyne.CanvasObject {
			check := widget.NewCheck("", nil)
			check.Hide()
			checks[check] = ""
			check.OnChanged = func(checked bool) {
				if !suppress[check] {
					b.setPathSelected(checks[check], checked)
				}
			}
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
			content := container.New(iconCellLayout{size: fyne.NewSize(sz.cellW, sz.cellH)}, check, imgArea, nameLabel)
			return content
		},
		func(id widget.GridWrapItemID, obj fyne.CanvasObject) {
			content := obj.(*fyne.Container)
			check := content.Objects[0].(*widget.Check)
			imgArea := content.Objects[1].(*fyne.Container)
			img := imgArea.Objects[0].(*fyne.Container).Objects[0].(*canvas.Image)
			loading := imgArea.Objects[1].(*fyne.Container).Objects[0].(*widget.Label)
			nameLabel := content.Objects[2].(*widget.Label)

			if id < 0 || id >= len(b.entries) {
				checks[check] = ""
				check.Hide()
				nameLabel.SetText("")
				loading.Hide()
				return
			}
			entry := b.entries[id]
			if b.multi && isRegularFileEntry(entry) {
				checks[check] = entry.Path
				suppress[check] = true
				check.SetChecked(b.selectedRows[entry.Path])
				suppress[check] = false
				check.Show()
			} else {
				checks[check] = ""
				check.Hide()
			}
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
