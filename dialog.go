package gofiledialog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

const settingsSaveDelay = 500 * time.Millisecond

// fyneLastFolderKey is the key used by Fyne's built-in file dialogs. Reading
// it lets applications switch between Fyne's dialog and this one without
// losing their most recently used folder.
const fyneLastFolderKey = "fyne:fileDialogLastFolder"

// Option configures a dialog created by NewOpen (and, in later phases,
// NewSave / NewFolder).
type Option func(*options)

type options struct {
	title    string
	startDir string
	store    Store
	filters  []Filter
	multi    bool
	fileName string
}

// WithTitle sets the dialog window's title.
func WithTitle(title string) Option {
	return func(o *options) { o.title = title }
}

// WithStartDir sets the directory the dialog opens to.
func WithStartDir(dir string) Option {
	return func(o *options) { o.startDir = dir }
}

// WithStore overrides where the last folder and view/sort/column/window
// settings are persisted. By default they're shared, via a JSON file in the
// OS config directory, across every app on the machine using gofiledialog's
// default settings.
func WithStore(store Store) Option {
	return func(o *options) { o.store = store }
}

// WithFilters sets the file-type choices shown in the Type dropdown. The
// first filter is selected initially. Directories are never filtered out.
func WithFilters(filters ...Filter) Option {
	return func(o *options) { o.filters = append([]Filter(nil), filters...) }
}

// WithMultiSelect allows Open dialogs to accumulate multiple selected files.
// Fyne's built-in collection widgets do not expose modifier-key selection, so
// selections accumulate as files are clicked.
func WithMultiSelect(enabled bool) Option {
	return func(o *options) { o.multi = enabled }
}

// WithFileName pre-fills the Save dialog filename entry.
func WithFileName(name string) Option {
	return func(o *options) { o.fileName = name }
}

func defaultStartDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return "."
}

// startDir selects the initial directory. An explicit option always wins;
// otherwise prefer Fyne's remembered file-dialog directory, then this
// package's persisted directory, and finally the user's home directory.
func startDir(explicit string, settings Settings) string {
	if explicit != "" {
		return explicit
	}
	if app := fyne.CurrentApp(); app != nil {
		if uri, err := storage.ParseURI(app.Preferences().String(fyneLastFolderKey)); err == nil && uri != nil && uri.Scheme() == "file" {
			if path := filepath.FromSlash(uri.Path()); isDirectory(path) {
				return path
			}
		}
	}
	if isDirectory(settings.LastDir) {
		return settings.LastDir
	}
	return defaultStartDir()
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// OpenDialog is a file-open dialog window.
type OpenDialog struct {
	win        fyne.Window
	browser    *Browser
	fileEntry  *widget.Entry
	filter     Filter
	onChosen   func(paths []string, err error)
	completion dialogCompletion
}

// SaveDialog is a file-save dialog window.
type SaveDialog struct {
	win        fyne.Window
	browser    *Browser
	fileEntry  *widget.Entry
	filter     Filter
	onChosen   func(paths []string, err error)
	completion dialogCompletion
}

// FolderDialog is a folder-selection dialog window.
type FolderDialog struct {
	win        fyne.Window
	browser    *Browser
	onChosen   func(paths []string, err error)
	completion dialogCompletion
}

type dialogCompletion struct {
	once sync.Once
}

func (c *dialogCompletion) finish(win fyne.Window, closeWindow bool, onChosen func([]string, error), paths []string, err error) {
	var completed bool
	var callbackPaths []string
	var callbackErr error
	var callback func([]string, error)
	c.once.Do(func() {
		completed = true
		callback = onChosen
		callbackErr = err
		if paths != nil {
			callbackPaths = append([]string(nil), paths...)
		}
	})
	if !completed {
		return
	}
	if closeWindow && win != nil {
		win.Close()
	}
	if callback != nil {
		callback(callbackPaths, callbackErr)
	}
}

// NewOpen builds an Open-file dialog. The dialog is not shown until Show is
// called. parent is accepted for API compatibility with Fyne's dialog helpers;
// gofiledialog currently creates a top-level window and centers it on screen
// because fyne.Window does not expose portable parent-relative placement.
func NewOpen(parent fyne.Window, opts ...Option) (*OpenDialog, error) {
	o := options{title: "Open"}
	for _, opt := range opts {
		opt(&o)
	}
	o.filters = normalizeFilters(o.filters)
	if err := validateFilterLabels(o.filters); err != nil {
		return nil, err
	}
	if o.store == nil {
		o.store = newFileStore()
	}
	settings, _ := o.store.Load()
	if settings.WindowWidth <= 0 || settings.WindowHeight <= 0 {
		settings.WindowWidth, settings.WindowHeight = 720, 480
	}
	o.startDir = startDir(o.startDir, settings)

	browser, err := NewBrowser(o.startDir)
	if err != nil {
		return nil, err
	}
	browser.applySettings(settings)
	browser.SetMultiSelect(o.multi)
	activeFilter := o.filters[0]
	if err := browser.SetFilter(activeFilter.Match); err != nil {
		return nil, err
	}

	d := &OpenDialog{browser: browser, filter: activeFilter}
	win := fyne.CurrentApp().NewWindow(o.title)
	d.win = win

	showErr := func(err error) { dialog.ShowError(err, win) }
	browser.OnError = showErr

	saver := newDebouncedSaver(o.store, settingsSaveDelay)
	buildSettings := func() Settings { return snapshotSettings(browser, win) }
	browser.OnSettingsChanged = func() { saver.Save(buildSettings()) }
	win.SetOnClosed(func() {
		saver.Flush(buildSettings())
		browser.Close()
		d.finishFromWindowClose()
	})

	newFolderBtn := widget.NewButton("New Folder", func() { showNewFolderDialog(win, browser, showErr) })
	toolbar := container.NewVBox(newToolbar(browser, showErr), container.NewHBox(newFolderBtn))
	columnsMenu := newColumnsMenu(win, browser)
	if browser.ViewMode() != ViewDetails {
		columnsMenu.Disable()
	}
	viewSwitcher := newViewSwitcher(browser, func(mode ViewMode) {
		if mode == ViewDetails {
			columnsMenu.Enable()
		} else {
			columnsMenu.Disable()
		}
	})
	rightControls := container.NewHBox(viewSwitcher, columnsMenu)
	topBar := container.NewBorder(nil, nil, nil, rightControls, toolbar)
	sidebar := newPlacesSidebar(browser, showErr)

	fileNameEntry := widget.NewEntry()
	fileNameEntry.SetPlaceHolder("File name")
	d.fileEntry = fileNameEntry
	filterSelect := newFilterSelect(o.filters, activeFilter, func(filter Filter) {
		activeFilter = filter
		d.filter = filter
		if err := browser.SetFilter(filter.Match); err != nil {
			showErr(err)
		}
	})
	browser.OnSelectionChanged = func() {
		selected := browser.SelectedEntries()
		names := make([]string, 0, len(selected))
		for _, entry := range selected {
			names = append(names, entry.Name)
		}
		fileNameEntry.SetText(strings.Join(names, "; "))
	}
	openBtn := widget.NewButton("Open", func() { d.chooseSelected() })
	cancelBtn := widget.NewButton("Cancel", func() { d.cancel() })
	buttons := container.NewHBox(layout.NewSpacer(), cancelBtn, openBtn)
	fields := container.NewGridWithColumns(2,
		container.NewBorder(nil, nil, widget.NewLabel("File name:"), nil, fileNameEntry),
		container.NewBorder(nil, nil, widget.NewLabel("Type:"), nil, filterSelect),
	)
	bottom := container.NewVBox(fields, buttons)

	browser.OnActivate = func(entry FileEntry) {
		d.finish([]string{entry.Path}, nil)
	}
	installKeyboardShortcuts(win, browser, d.chooseSelected, d.cancel)

	main := container.NewBorder(topBar, bottom, sidebar, nil, browser.Content())
	win.SetContent(main)
	win.Resize(fyne.NewSize(settings.WindowWidth, settings.WindowHeight))

	return d, nil
}

// NewSave builds a Save-file dialog. The chosen path is returned without
// creating the file; callers decide how to write it. parent is accepted for API
// compatibility with Fyne's dialog helpers; gofiledialog currently creates a
// top-level window and centers it on screen because fyne.Window does not expose
// portable parent-relative placement.
func NewSave(parent fyne.Window, opts ...Option) (*SaveDialog, error) {
	o := options{title: "Save As"}
	for _, opt := range opts {
		opt(&o)
	}
	o.filters = normalizeFilters(o.filters)
	if err := validateFilterLabels(o.filters); err != nil {
		return nil, err
	}
	if o.store == nil {
		o.store = newFileStore()
	}
	settings, _ := o.store.Load()
	o.startDir = startDir(o.startDir, settings)

	browser, err := NewBrowser(o.startDir)
	if err != nil {
		return nil, err
	}
	browser.applySettings(settings)
	activeFilter := o.filters[0]
	if err := browser.SetFilter(activeFilter.Match); err != nil {
		return nil, err
	}

	d := &SaveDialog{browser: browser, filter: activeFilter}
	win := fyne.CurrentApp().NewWindow(o.title)
	d.win = win
	showErr := func(err error) { dialog.ShowError(err, win) }
	browser.OnError = showErr

	saver := newDebouncedSaver(o.store, settingsSaveDelay)
	buildSettings := func() Settings { return snapshotSettings(browser, win) }
	browser.OnSettingsChanged = func() { saver.Save(buildSettings()) }
	win.SetOnClosed(func() {
		saver.Flush(buildSettings())
		browser.Close()
		d.finishFromWindowClose()
	})

	newFolderBtn := widget.NewButton("New Folder", func() { showNewFolderDialog(win, browser, showErr) })
	toolbar := container.NewVBox(newToolbar(browser, showErr), container.NewHBox(newFolderBtn))
	columnsMenu := newColumnsMenu(win, browser)
	if browser.ViewMode() != ViewDetails {
		columnsMenu.Disable()
	}
	viewSwitcher := newViewSwitcher(browser, func(mode ViewMode) {
		if mode == ViewDetails {
			columnsMenu.Enable()
		} else {
			columnsMenu.Disable()
		}
	})
	topBar := container.NewBorder(nil, nil, nil, container.NewHBox(viewSwitcher, columnsMenu), toolbar)
	sidebar := newPlacesSidebar(browser, showErr)

	fileNameEntry := widget.NewEntry()
	fileNameEntry.SetPlaceHolder("File name")
	fileNameEntry.SetText(o.fileName)
	d.fileEntry = fileNameEntry
	filterSelect := newFilterSelect(o.filters, activeFilter, func(filter Filter) {
		activeFilter = filter
		d.filter = filter
		fileNameEntry.SetText(ensureExtension(fileNameEntry.Text, filter))
		if err := browser.SetFilter(filter.Match); err != nil {
			showErr(err)
		}
	})
	browser.OnSelectionChanged = func() {
		if entry, ok := browser.Selected(); ok && !entry.IsDir {
			fileNameEntry.SetText(entry.Name)
		}
	}
	browser.OnActivate = func(entry FileEntry) {
		if !entry.IsDir {
			fileNameEntry.SetText(entry.Name)
			d.chooseSave()
		}
	}
	fileNameEntry.OnSubmitted = func(string) { d.chooseSave() }
	installKeyboardShortcuts(win, browser, d.chooseSave, d.cancel)

	saveBtn := widget.NewButton("Save", func() { d.chooseSave() })
	cancelBtn := widget.NewButton("Cancel", func() { d.cancel() })
	buttons := container.NewHBox(layout.NewSpacer(), cancelBtn, saveBtn)
	fields := container.NewGridWithColumns(2,
		container.NewBorder(nil, nil, widget.NewLabel("File name:"), nil, fileNameEntry),
		container.NewBorder(nil, nil, widget.NewLabel("Type:"), nil, filterSelect),
	)

	main := container.NewBorder(topBar, container.NewVBox(fields, buttons), sidebar, nil, browser.Content())
	win.SetContent(main)
	win.Resize(fyne.NewSize(settings.WindowWidth, settings.WindowHeight))
	return d, nil
}

// NewFolder builds a folder-selection dialog. The currently displayed folder
// is returned when the user chooses Select Folder. parent is accepted for API
// compatibility with Fyne's dialog helpers; gofiledialog currently creates a
// top-level window and centers it on screen because fyne.Window does not expose
// portable parent-relative placement.
func NewFolder(parent fyne.Window, opts ...Option) (*FolderDialog, error) {
	o := options{title: "Select Folder"}
	for _, opt := range opts {
		opt(&o)
	}
	if o.store == nil {
		o.store = newFileStore()
	}
	settings, _ := o.store.Load()
	o.startDir = startDir(o.startDir, settings)

	browser, err := NewBrowser(o.startDir)
	if err != nil {
		return nil, err
	}
	browser.applySettings(settings)

	d := &FolderDialog{browser: browser}
	win := fyne.CurrentApp().NewWindow(o.title)
	d.win = win
	showErr := func(err error) { dialog.ShowError(err, win) }
	browser.OnError = showErr

	saver := newDebouncedSaver(o.store, settingsSaveDelay)
	buildSettings := func() Settings { return snapshotSettings(browser, win) }
	browser.OnSettingsChanged = func() { saver.Save(buildSettings()) }
	win.SetOnClosed(func() {
		saver.Flush(buildSettings())
		browser.Close()
		d.finishFromWindowClose()
	})

	newFolderBtn := widget.NewButton("New Folder", func() { showNewFolderDialog(win, browser, showErr) })
	toolbar := container.NewVBox(newToolbar(browser, showErr), container.NewHBox(newFolderBtn))
	columnsMenu := newColumnsMenu(win, browser)
	if browser.ViewMode() != ViewDetails {
		columnsMenu.Disable()
	}
	viewSwitcher := newViewSwitcher(browser, func(mode ViewMode) {
		if mode == ViewDetails {
			columnsMenu.Enable()
		} else {
			columnsMenu.Disable()
		}
	})
	topBar := container.NewBorder(nil, nil, nil, container.NewHBox(viewSwitcher, columnsMenu), toolbar)
	sidebar := newPlacesSidebar(browser, showErr)

	selectBtn := widget.NewButton("Select Folder", func() { d.finish([]string{browser.CurrentDir()}, nil) })
	cancelBtn := widget.NewButton("Cancel", func() { d.cancel() })
	installKeyboardShortcuts(win, browser, func() { d.finish([]string{browser.CurrentDir()}, nil) }, d.cancel)
	buttons := container.NewHBox(layout.NewSpacer(), cancelBtn, selectBtn)

	main := container.NewBorder(topBar, buttons, sidebar, nil, browser.Content())
	win.SetContent(main)
	win.Resize(fyne.NewSize(settings.WindowWidth, settings.WindowHeight))
	return d, nil
}

// snapshotSettings captures the current persisted-state fields from browser
// and win into a Settings value ready to save.
func snapshotSettings(browser *Browser, win fyne.Window) Settings {
	cols := browser.Columns()
	colSettings := make([]ColumnSetting, len(cols))
	for i, c := range cols {
		colSettings[i] = ColumnSetting{ID: c.ID, Visible: c.Visible, Width: c.Width}
	}
	size := win.Canvas().Size()
	return Settings{
		LastDir:       browser.CurrentDir(),
		ViewMode:      string(browser.ViewMode()),
		ShowHidden:    browser.ShowHidden(),
		SortColumn:    browser.SortColumn(),
		SortAscending: browser.SortAscending(),
		Columns:       colSettings,
		WindowWidth:   size.Width,
		WindowHeight:  size.Height,
	}
}

func (d *OpenDialog) chooseSelected() {
	if names := strings.TrimSpace(d.fileEntry.Text); names != "" {
		paths, err := d.typedPaths(names)
		if err != nil {
			dialog.ShowError(err, d.win)
			return
		}
		d.finish(paths, nil)
		return
	}
	selected := d.browser.SelectedEntries()
	if len(selected) == 0 {
		return
	}
	paths := make([]string, 0, len(selected))
	for _, entry := range selected {
		paths = append(paths, entry.Path)
	}
	d.finish(paths, nil)
}

// typedPaths resolves semicolon-separated names from the displayed directory.
// It accepts only regular files that match the active filter.
func (d *OpenDialog) typedPaths(names string) ([]string, error) {
	parts := strings.Split(names, ";")
	if len(parts) > 1 && !d.browser.MultiSelect() {
		return nil, fmt.Errorf("multiple files are not allowed")
	}
	paths := make([]string, 0, len(parts))
	for _, name := range parts {
		name = strings.TrimSpace(name)
		if name == "" || filepath.Base(name) != name || name == "." {
			return nil, fmt.Errorf("invalid file name %q", name)
		}
		path := filepath.Join(d.browser.CurrentDir(), name)
		info, err := os.Stat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("file %q does not exist", name)
			}
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%q is not a regular file", name)
		}
		entry := FileEntry{Name: name, Path: path}
		if !d.filter.Match(entry) {
			return nil, fmt.Errorf("file %q does not match the selected type", name)
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func (d *OpenDialog) cancel() {
	d.finish(nil, nil)
}

func (d *OpenDialog) finish(paths []string, err error) {
	d.completion.finish(d.win, true, d.onChosen, paths, err)
}

func (d *OpenDialog) finishFromWindowClose() {
	d.completion.finish(d.win, false, d.onChosen, nil, nil)
}

// SetOnChosen sets the callback invoked when the dialog closes. paths is nil
// if the user cancelled.
func (d *OpenDialog) SetOnChosen(f func(paths []string, err error)) {
	d.onChosen = f
}

// Show displays the dialog window, centered on screen.
func (d *OpenDialog) Show() {
	d.win.CenterOnScreen()
	d.win.Show()
}

func (d *SaveDialog) chooseSave() {
	path, err := savePath(d.browser.CurrentDir(), d.fileEntry.Text, d.filter)
	if err != nil {
		dialog.ShowError(err, d.win)
		return
	}
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			dialog.ShowError(fmt.Errorf("%q is a directory", filepath.Base(path)), d.win)
			return
		}
		dialog.ShowConfirm("Overwrite?", "Are you sure you want to overwrite "+filepath.Base(path)+"?", func(ok bool) {
			if ok {
				d.finish([]string{path}, nil)
			}
		}, d.win)
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		dialog.ShowError(err, d.win)
		return
	}
	d.finish([]string{path}, nil)
}

func (d *SaveDialog) cancel() {
	d.finish(nil, nil)
}

func (d *SaveDialog) finish(paths []string, err error) {
	d.completion.finish(d.win, true, d.onChosen, paths, err)
}

func (d *SaveDialog) finishFromWindowClose() {
	d.completion.finish(d.win, false, d.onChosen, nil, nil)
}

// SetOnChosen sets the callback invoked when the Save dialog closes.
func (d *SaveDialog) SetOnChosen(f func(paths []string, err error)) {
	d.onChosen = f
}

// Show displays the Save dialog window, centered on screen.
func (d *SaveDialog) Show() {
	d.win.CenterOnScreen()
	d.win.Show()
}

func (d *FolderDialog) cancel() {
	d.finish(nil, nil)
}

func (d *FolderDialog) finish(paths []string, err error) {
	d.completion.finish(d.win, true, d.onChosen, paths, err)
}

func (d *FolderDialog) finishFromWindowClose() {
	d.completion.finish(d.win, false, d.onChosen, nil, nil)
}

// SetOnChosen sets the callback invoked when the Folder dialog closes.
func (d *FolderDialog) SetOnChosen(f func(paths []string, err error)) {
	d.onChosen = f
}

// Show displays the Folder dialog window, centered on screen.
func (d *FolderDialog) Show() {
	d.win.CenterOnScreen()
	d.win.Show()
}

// ShowOpen is a convenience wrapper that builds and immediately shows an
// Open dialog, in the style of fyne.io/fyne/v2/dialog.ShowFileOpen. parent is
// accepted for API compatibility but is not used for placement.
func ShowOpen(onChosen func(paths []string, err error), parent fyne.Window, opts ...Option) error {
	d, err := NewOpen(parent, opts...)
	if err != nil {
		return err
	}
	d.SetOnChosen(onChosen)
	d.Show()
	return nil
}

// ShowSave is a convenience wrapper that builds and immediately shows a Save
// dialog. parent is accepted for API compatibility but is not used for
// placement.
func ShowSave(onChosen func(paths []string, err error), parent fyne.Window, opts ...Option) error {
	d, err := NewSave(parent, opts...)
	if err != nil {
		return err
	}
	d.SetOnChosen(onChosen)
	d.Show()
	return nil
}

// ShowFolder is a convenience wrapper that builds and immediately shows a
// folder-selection dialog. parent is accepted for API compatibility but is not
// used for placement.
func ShowFolder(onChosen func(paths []string, err error), parent fyne.Window, opts ...Option) error {
	d, err := NewFolder(parent, opts...)
	if err != nil {
		return err
	}
	d.SetOnChosen(onChosen)
	d.Show()
	return nil
}

func normalizeFilters(filters []Filter) []Filter {
	if len(filters) == 0 {
		return []Filter{{Name: "All files"}}
	}
	return append([]Filter(nil), filters...)
}

func newFilterSelect(filters []Filter, active Filter, onChange func(Filter)) *widget.Select {
	labels := make([]string, len(filters))
	byLabel := make(map[string]Filter, len(filters))
	for i, filter := range filters {
		label := filter.Label()
		labels[i] = label
		byLabel[label] = filter
	}
	selectWidget := widget.NewSelect(labels, func(label string) {
		if onChange != nil {
			onChange(byLabel[label])
		}
	})
	selectWidget.SetSelected(active.Label())
	return selectWidget
}

func savePath(dir, name string, filter Filter) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("enter a file name")
	}
	name = ensureExtension(name, filter)
	if filepath.IsAbs(name) {
		return filepath.Clean(name), nil
	}
	return filepath.Join(dir, name), nil
}

func showNewFolderDialog(win fyne.Window, b *Browser, onError func(error)) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Folder name")
	dialog.ShowForm("New Folder", "Create", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
	}, func(ok bool) {
		if !ok {
			return
		}
		path, err := newFolderPath(b.CurrentDir(), nameEntry.Text)
		if err != nil {
			if onError != nil {
				onError(err)
			}
			return
		}
		if err := os.Mkdir(path, 0o755); err != nil {
			if onError != nil {
				onError(err)
			}
			return
		}
		if err := b.Refresh(); err != nil && onError != nil {
			onError(err)
		}
	}, win)
}

func newFolderPath(dir, name string) (string, error) {
	if strings.TrimSpace(name) != name {
		if strings.TrimSpace(name) == "" {
			return "", errors.New("enter a folder name")
		}
		return "", errors.New("folder name must not start or end with whitespace")
	}
	if err := validateFolderName(name); err != nil {
		return "", err
	}

	baseDir := filepath.Clean(dir)
	path := filepath.Join(baseDir, name)
	cleaned := filepath.Clean(path)
	rel, err := filepath.Rel(baseDir, cleaned)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", errors.New("folder name must stay inside the current folder")
	}
	return cleaned, nil
}

func validateFolderName(name string) error {
	if name == "" {
		return errors.New("enter a folder name")
	}
	if name == "." || name == ".." {
		return errors.New("folder name must not be . or ..")
	}
	if filepath.IsAbs(name) {
		return errors.New("folder name must not be an absolute path")
	}
	if strings.ContainsAny(name, `/\`) {
		return errors.New("folder name must not contain path separators")
	}
	if strings.IndexFunc(name, func(r rune) bool { return r == 0 || r < 32 }) >= 0 {
		return errors.New("folder name contains an invalid character")
	}
	if err := validatePlatformFolderName(name); err != nil {
		return err
	}
	return nil
}

func installKeyboardShortcuts(win fyne.Window, b *Browser, onEnter, onCancel func()) {
	var typeAhead string
	var lastTyped time.Time

	win.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		switch ev.Name {
		case fyne.KeyDown:
			b.MoveSelection(1)
			return
		case fyne.KeyUp:
			b.MoveSelection(-1)
			return
		case fyne.KeyReturn, fyne.KeyEnter:
			if entry, ok := b.Selected(); ok && entry.IsDir {
				b.ActivateSelected()
				return
			}
			if onEnter != nil {
				onEnter()
			}
			return
		case fyne.KeyBackspace:
			if err := b.Up(); err != nil && b.OnError != nil {
				b.OnError(err)
			}
			return
		case fyne.KeyEscape:
			if onCancel != nil {
				onCancel()
			}
			return
		}

		key := string(ev.Name)
		if len([]rune(key)) != 1 {
			return
		}
		if time.Since(lastTyped) > time.Second {
			typeAhead = ""
		}
		lastTyped = time.Now()
		typeAhead += key
		if !b.TypeAhead(typeAhead) && len(typeAhead) > 1 {
			typeAhead = key
			b.TypeAhead(typeAhead)
		}
	})
}
