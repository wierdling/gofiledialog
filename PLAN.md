# Plan: `gofiledialog` — a Windows-Explorer-class file dialog package for Fyne

## Goal

A reusable Go package providing Open File, Save File, and Select Folder dialogs for
Fyne apps, with features comparable to the native Windows dialog:

- Switchable views: **Details, List, Small icons, Medium icons, Large icons**
- **Sortable columns** in Details view (click header to sort, click again to reverse)
- **Configurable columns** (choose which columns show; Name is always on)
- **Settings persist across apps**: view mode, sort column/direction, visible
  columns and widths are saved to a shared config file and reapplied next time
  *any* of your apps opens the dialog
- **Real image thumbnails** in the icon views, loaded asynchronously with caching

Module path: `github.com/wierdling/gofiledialog`
Target: Fyne v2 (latest, currently v2.6.x), Go 1.22+. Cross-platform, Windows-first.

## Why not extend Fyne's built-in dialog

`fyne.io/fyne/v2/dialog.FileDialog` keeps its internals unexported and offers no
hooks for views, columns, or sorting. We build our own dialog window from Fyne
primitives instead; the public API will *feel* like Fyne's (`ShowFileOpen`-style
callbacks) so it's a drop-in habit swap.

## Public API sketch

```go
import "github.com/wierdling/gofiledialog"

// Simple, Fyne-style:
gofiledialog.ShowOpen(func(paths []string, err error) { ... }, parentWindow)

// Configured:
d := gofiledialog.NewOpen(parentWindow,
    gofiledialog.WithFilters(
        gofiledialog.Filter{Name: "Images", Extensions: []string{".png", ".jpg", ".gif"}},
        gofiledialog.Filter{Name: "All files", Extensions: nil},
    ),
    gofiledialog.WithStartDir(`D:\photos`),
    gofiledialog.WithMultiSelect(true),
    gofiledialog.WithTitle("Choose images"),
)
d.SetOnChosen(func(paths []string, err error) { ... })
d.Show()

// Also: gofiledialog.NewSave(...), gofiledialog.NewFolder(...)
```

Options are functional (`With...`) so new features never break callers.

## Dialog layout (mirrors Explorer)

```
┌──────────────────────────────────────────────────────────────┐
│ [◀][▶][▲]  [breadcrumb / editable path bar]   [🔍 filter]    │
│ [New Folder]                       [View ▾] [Columns ▾]      │
├──────────┬───────────────────────────────────────────────────┤
│ Places   │  File browser area                                │
│  Home    │   Details  → widget.Table with clickable headers  │
│  Desktop │   Icons    → widget.GridWrap (3 cell sizes)       │
│  Docs    │   List     → widget.List                          │
│  Downloads│                                                  │
│  ─drives─│                                                   │
│  C:\ D:\ │                                                   │
├──────────┴───────────────────────────────────────────────────┤
│ File name: [____________________]  Type: [Images (*.png) ▾]  │
│                                        [Open]     [Cancel]   │
└──────────────────────────────────────────────────────────────┘
```

## Package layout

```
gofiledialog/
├── go.mod
├── dialog.go            // public API: NewOpen/NewSave/NewFolder, options, Show
├── browser.go           // core browser widget: toolbar + sidebar + view host
├── entry.go             // FileEntry model: name, size, mod/created time, kind, isDir
├── list_dir.go          // directory reading, filtering, hidden-file logic
├── sort.go              // sort by any column, asc/desc, dirs-first
├── columns.go           // column registry: id, title, width, formatter, visible
├── view_details.go      // widget.Table view, header buttons w/ sort arrows
├── view_icons.go        // widget.GridWrap view, small/medium/large cell presets
├── view_list.go         // compact list view
├── navigation.go        // history (back/fwd/up), breadcrumb bar, places sidebar
├── places_windows.go    // drive enumeration (GetLogicalDrives), known folders
├── places_other.go      // unix mounts, XDG dirs
├── ctime_windows.go     // creation time via FileInfo.Sys().(*syscall.Win32FileAttributeData)
├── ctime_darwin.go      // Birthtimespec
├── ctime_linux.go       // best-effort (statx birth time; falls back to zero/hidden)
├── settings.go          // shared JSON settings: load/save/debounce
├── icons.go             // extension → file-type icon mapping (theme-aware)
├── thumbnail.go         // async thumbnail pipeline: worker pool + LRU + disk cache
├── internal/lru/        // small LRU cache
└── cmd/demo/main.go     // runnable demo app exercising all three dialogs
```

## Key technical decisions

### Details view: sortable, resizable, choosable columns
- `widget.Table` with `ShowHeaderRow = true`; `CreateHeader`/`UpdateHeader`
  return tappable header cells showing the column title plus a ▲/▼ sort indicator.
- Built-in columns: **Name** (always on), **Size**, **Type**, **Date modified**,
  **Date created**. Registry design makes adding more (e.g. extension) trivial.
- Column widths set via `Table.SetColumnWidth`; user resizing isn't native to
  Fyne tables, so v1 persists widths from settings + sensible autosize, with a
  drag-to-resize enhancement as a stretch goal.
- **Columns ▾** toolbar menu = checkbox list toggling visibility (Explorer's
  right-click-header equivalent; Fyne has no header context menu, so a toolbar
  menu is the reliable cross-platform spot).

### Created date, cross-platform
Go's `os.FileInfo` only guarantees ModTime. Creation time comes from
`FileInfo.Sys()` per platform (build-tagged files). On Linux where birth time
is unavailable, the Date-created column shows “—” and is hidden by default.

### Icon views + thumbnails
- One `widget.GridWrap` reused for Small/Medium/Large with different cell and
  icon sizes (48 / 96 / 160 px class).
- Thumbnails: bounded worker pool (≈ NumCPU) decodes and downscales images
  (`image/png`, `jpeg`, `gif`; `golang.org/x/image` adds bmp/tiff/webp).
  Results go into an in-memory LRU keyed by `path+modtime+size`, plus a disk
  cache under the config dir so reopening a folder is instant. Cells show the
  file-type icon immediately and refresh when the thumbnail lands. Scrolled-away
  requests are cancellable.

### Shared persistent settings
- JSON at `os.UserConfigDir()/wierdling-gofiledialog/settings.json`
  (Windows: `%AppData%\wierdling-gofiledialog\settings.json`).
- Persisted: view mode, sort column + direction, visible columns + order +
  widths, dialog size, show-hidden flag, sidebar width. (Last directory stays
  per-app via an option, not shared — different apps want different folders.)
- Written debounced on change and on dialog close; corrupt/missing file falls
  back to defaults silently. All access behind a tiny `Store` interface so an
  app *could* inject its own storage later.

### Explorer-parity behaviors
- Back / Forward / Up with history; breadcrumb that flips to an editable path
  entry on click; type a path + Enter to jump.
- Places sidebar: Home, Desktop, Documents, Downloads, Pictures + all drives
  (Windows) / mounts (unix).
- Double-click dir = enter; double-click file = choose. Enter key opens,
  Backspace goes up, type-ahead jumps to matching name.
- File-type filter dropdown; multi-select (Ctrl/Shift) when enabled.
- Save dialog: filename pre-fill, extension enforcement, overwrite confirmation.
- New Folder button; show/hidden toggle (dotfiles + Windows hidden attribute).

## Implementation phases

Each phase ends with the demo app runnable.

1. **Scaffold + core browsing** — repo, `go.mod`, demo app; dialog window with
   Details view (fixed columns), directory listing, double-click navigation,
   Open/Cancel wiring for the Open dialog.
2. **Navigation shell** — sidebar places + drives, back/forward/up history,
   breadcrumb/editable path bar, hidden-files toggle.
3. **Details view, full** — sortable headers with indicators, created-time
   per-platform support, column registry + Columns menu, formatted sizes/dates.
4. **Persistence** — settings store, load-on-open / save-on-change, applied
   across view mode, sort, columns, window size.
5. **Icon & list views + thumbnails** — view switcher, GridWrap views, async
   thumbnail pipeline with memory + disk cache.
6. **Save & Folder dialogs + selection features** — filters dropdown,
   multi-select, New Folder, overwrite confirm, filename entry logic.
7. **Polish & release** — keyboard nav + type-ahead, unit tests (sorting,
   settings, filters, thumbnail cache keys), README with screenshots, GoDoc,
   tag `v0.1.0`, verify `go get` from a second project.

## Confirmed

- GitHub username: `wierdling`. Package/repo name: `gofiledialog`.
- Module path: `github.com/wierdling/gofiledialog`.
