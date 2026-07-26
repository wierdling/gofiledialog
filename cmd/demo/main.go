// Command demo exercises the gofiledialog dialogs.
package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/wierdling/gofiledialog"
)

func main() {
	a := app.New()
	win := a.NewWindow("gofiledialog demo")

	result := widget.NewLabel("No file chosen yet. In Open, check multiple files, then press Open.")
	result.Wrapping = fyne.TextWrapWord

	openBtn := widget.NewButton("Open files (check multiple)...", func() {
		gofiledialog.ShowOpen(func(paths []string, err error) {
			switch {
			case err != nil:
				result.SetText(fmt.Sprintf("Error: %v", err))
			case len(paths) == 0:
				result.SetText("Cancelled.")
			default:
				result.SetText("Chosen: " + strings.Join(paths, ", "))
			}
		}, win,
			gofiledialog.WithFilters(
				gofiledialog.Filter{Name: "Images", Extensions: []string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".tif", ".tiff", ".webp"}},
				gofiledialog.Filter{Name: "All files"},
			),
			gofiledialog.WithMultiSelect(true),
		)
	})

	saveBtn := widget.NewButton("Save file...", func() {
		gofiledialog.ShowSave(func(paths []string, err error) {
			switch {
			case err != nil:
				result.SetText(fmt.Sprintf("Error: %v", err))
			case len(paths) == 0:
				result.SetText("Cancelled.")
			default:
				result.SetText("Save path: " + strings.Join(paths, ", "))
			}
		}, win,
			gofiledialog.WithFileName("untitled.txt"),
			gofiledialog.WithFilters(
				gofiledialog.Filter{Name: "Text files", Extensions: []string{".txt"}},
				gofiledialog.Filter{Name: "All files"},
			),
		)
	})

	folderBtn := widget.NewButton("Select folder...", func() {
		gofiledialog.ShowFolder(func(paths []string, err error) {
			switch {
			case err != nil:
				result.SetText(fmt.Sprintf("Error: %v", err))
			case len(paths) == 0:
				result.SetText("Cancelled.")
			default:
				result.SetText("Folder: " + strings.Join(paths, ", "))
			}
		}, win)
	})

	win.SetContent(container.NewVBox(openBtn, saveBtn, folderBtn, result))
	win.Resize(fyne.NewSize(380, 220))
	win.ShowAndRun()
}
