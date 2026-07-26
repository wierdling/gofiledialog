# gofiledialog

Explorer-style Open, Save, and Select Folder dialogs for Fyne applications.

`gofiledialog` is a reusable Go package for apps that need more than Fyne's
built-in file dialog exposes. It builds a dialog from Fyne primitives so it can
offer switchable views, sortable Details columns, persistent shared settings,
file-type filters, New Folder support, and asynchronous image thumbnails.

The public API follows the same shape as Fyne's dialog helpers: use `ShowOpen`,
`ShowSave`, or `ShowFolder` for simple cases, or construct a dialog with
`NewOpen`, `NewSave`, or `NewFolder` when you want to configure it before
showing it.

## Features

- Open, Save, and Select Folder dialogs
- Details, List, Small icons, Medium icons, and Large icons views
- Sortable Details columns with persisted column order, visibility, widths, and sort direction
- Shared settings across apps using the OS user config directory
- Places sidebar with Home, Desktop, Documents, Downloads, Pictures, and drives/mounts
- Back, Forward, Up, breadcrumb navigation, editable path entry, and hidden-file toggle
- File-type filters and Save default-extension handling
- New Folder creation and Save overwrite confirmation
- Async image thumbnails with bounded workers, memory cache, and disk cache
- Keyboard polish: Up/Down selection, Enter activation, Backspace up, Escape cancel, and type-ahead search

## Install

```sh
go get github.com/wierdling/gofiledialog
```

The package targets Fyne v2.7.4 and Go 1.22 or newer.

## Quick Start

```go
package main

import (
	"fmt"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"

	"github.com/wierdling/gofiledialog"
)

func main() {
	a := app.New()
	win := a.NewWindow("Example")

	btn := widget.NewButton("Open...", func() {
		_ = gofiledialog.ShowOpen(func(paths []string, err error) {
			if err != nil {
				fmt.Println("open failed:", err)
				return
			}
			if len(paths) == 0 {
				fmt.Println("cancelled")
				return
			}
			fmt.Println("chosen:", paths)
		}, win)
	})

	win.SetContent(btn)
	win.ShowAndRun()
}
```

## Open Dialog

```go
err := gofiledialog.ShowOpen(func(paths []string, err error) {
	if err != nil {
		// handle error
		return
	}
	if len(paths) == 0 {
		// cancelled
		return
	}
	// use paths
}, win,
	gofiledialog.WithTitle("Choose images"),
	gofiledialog.WithStartDir(`D:\photos`),
	gofiledialog.WithFilters(
		gofiledialog.Filter{Name: "Images", Extensions: []string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".tif", ".tiff", ".webp"}},
		gofiledialog.Filter{Name: "All files"},
	),
	gofiledialog.WithMultiSelect(true),
)
if err != nil {
	// dialog construction failed
}
```

### Cross-platform multi-select

Fyne 2.7.4's `List`, `Table`, and `GridWrap` APIs do not expose portable
Ctrl/Shift modifier state. When `WithMultiSelect(true)` is enabled,
`gofiledialog` therefore uses an explicit checkbox fallback: check each file
you want (the checkbox is shown beside the name in every view), uncheck files
to remove them, and press **Open** to submit the checked files in display
order. This works consistently across desktop and mobile backends and does
not require modifier keys. Directory rows remain navigational; only checked
regular files are returned.

## Save Dialog

`ShowSave` returns the chosen path; it does not create or write the file for
you. If the user enters a filename without an extension, the selected filter's
first extension is appended.

```go
err := gofiledialog.ShowSave(func(paths []string, err error) {
	if err != nil {
		return
	}
	if len(paths) == 0 {
		return // cancelled
	}

	path := paths[0]
	// write to path
}, win,
	gofiledialog.WithFileName("untitled.txt"),
	gofiledialog.WithFilters(
		gofiledialog.Filter{Name: "Text files", Extensions: []string{".txt"}},
		gofiledialog.Filter{Name: "All files"},
	),
)
```

## Folder Dialog

```go
err := gofiledialog.ShowFolder(func(paths []string, err error) {
	if err != nil {
		return
	}
	if len(paths) == 0 {
		return // cancelled
	}

	folder := paths[0]
	// use folder
}, win)
```

## Configured Dialogs

For more control, construct the dialog, set the callback, then show it:

```go
d, err := gofiledialog.NewOpen(win,
	gofiledialog.WithTitle("Import files"),
	gofiledialog.WithMultiSelect(true),
)
if err != nil {
	return err
}

d.SetOnChosen(func(paths []string, err error) {
	// handle result
})
d.Show()
```

## Options

- `WithTitle(title string)` sets the dialog window title.
- `WithStartDir(dir string)` sets the initial directory.
- `WithFilters(filters ...Filter)` configures the Type dropdown.
- `WithMultiSelect(enabled bool)` enables checkbox-based multi-select for Open
  dialogs. Check or uncheck files, then press Open to return the checked set.
- `WithFileName(name string)` pre-fills Save dialogs.
- `WithStore(store Store)` overrides the default shared settings store.

## Parent Window Parameter

The `parent fyne.Window` parameter on `ShowOpen`, `ShowSave`, `ShowFolder`,
`NewOpen`, `NewSave`, and `NewFolder` is accepted for API compatibility with
Fyne's dialog helpers. `gofiledialog` currently creates top-level windows and
centers them on screen because Fyne's public `fyne.Window` interface does not
expose portable parent-relative window placement.

## Settings

By default, view mode, sort column/direction, visible columns, column order,
column widths, dialog size, hidden-file visibility, and the last folder are persisted in:

```text
os.UserConfigDir()/wierdling-gofiledialog/settings.json
```

That means settings are shared by every app on the machine that uses the
default `gofiledialog` store. Use `WithStore` if an app needs isolated or
custom persistence. Settings persistence is best-effort: load failures fall
back to defaults, and save failures are ignored so a settings backend cannot
break a user's file-dialog interaction.

When no `WithStartDir` is supplied, the dialog first uses Fyne's saved
`fyne:fileDialogLastFolder` preference (when it names an accessible local
folder); otherwise it uses the saved `lastDir` setting.

The default store writes through a temporary file and atomically replaces the
settings file, so readers never observe partial JSON. Saves are serialized
within the process and coordinated across processes with a short-lived
`settings.json.lock` file. If another process holds the lock for too long, the
save is skipped and the last valid settings file is left intact.

## Thumbnails

Icon views show generic file-type icons immediately, then load real image
thumbnails asynchronously. Thumbnail work is bounded, prioritized by smaller
files first, cached in memory, and cached on disk under the OS user cache
directory.

Supported thumbnail formats include PNG, JPEG, GIF, BMP, TIFF, and WebP.

Large files and very high-resolution images are skipped so opening a folder
with many photos stays responsive.

## Demo

Run the demo:

```sh
go run ./cmd/demo
```

Build the demo:

```sh
go build -buildvcs=false -o demo.exe ./cmd/demo
```

Then run:

```powershell
.\demo.exe
```

## Development

Run tests:

```sh
go test ./...
```

Check package docs locally:

```sh
go doc .
```

## Status

This project is at the first release-pass stage for `v0.1.0`.

Multi-select intentionally uses checkboxes instead of Ctrl/Shift gestures so
the same interaction is available on every Fyne 2.7.4 backend.
