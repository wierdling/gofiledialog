package gofiledialog

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// tappableContainer wraps a canvas object to make it respond to taps,
// used so clicking empty space in the breadcrumb bar (not on a segment
// button) switches it into an editable path entry.
type tappableContainer struct {
	widget.BaseWidget
	content  fyne.CanvasObject
	onTapped func()
}

func newTappableContainer(content fyne.CanvasObject, onTapped func()) *tappableContainer {
	t := &tappableContainer{content: content, onTapped: onTapped}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappableContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.content)
}

func (t *tappableContainer) Tapped(*fyne.PointEvent) {
	if t.onTapped != nil {
		t.onTapped()
	}
}

// pathSegments splits an absolute path into clickable breadcrumb segments,
// each carrying the cumulative path up to and including that segment.
func pathSegments(path string) []Place {
	vol := filepath.VolumeName(path)
	rest := strings.TrimPrefix(path, vol)
	rest = strings.Trim(rest, string(filepath.Separator))

	rootName := vol
	cur := vol + string(filepath.Separator)
	if vol == "" {
		rootName = string(filepath.Separator)
		cur = string(filepath.Separator)
	}
	segments := []Place{{Name: rootName, Path: cur}}

	if rest == "" {
		return segments
	}
	for _, part := range strings.Split(rest, string(filepath.Separator)) {
		cur = filepath.Join(cur, part)
		segments = append(segments, Place{Name: part, Path: cur})
	}
	return segments
}

// newToolbar builds the back/forward/up buttons and the breadcrumb/path bar,
// wired to b. Errors from navigation attempts are reported via onError.
func newToolbar(b *Browser, onError func(error)) fyne.CanvasObject {
	navigate := func(err error) {
		if err != nil && onError != nil {
			onError(err)
		}
	}

	backBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() { navigate(b.Back()) })
	fwdBtn := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() { navigate(b.Forward()) })
	upBtn := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() { navigate(b.Up()) })
	hiddenCheck := widget.NewCheck("Hidden items", func(checked bool) { navigate(b.ToggleShowHidden(checked)) })
	hiddenCheck.SetChecked(b.ShowHidden())

	breadcrumbRow := container.NewHBox()
	pathEntry := widget.NewEntry()
	pathEntry.Hide()

	var breadcrumbTappable *tappableContainer
	switchToBreadcrumb := func() {
		pathEntry.Hide()
		breadcrumbTappable.Show()
	}
	switchToEdit := func() {
		pathEntry.SetText(b.CurrentDir())
		breadcrumbTappable.Hide()
		pathEntry.Show()
	}
	breadcrumbTappable = newTappableContainer(breadcrumbRow, switchToEdit)

	pathEntry.OnSubmitted = func(text string) {
		if err := b.NavigateTo(text); err != nil {
			navigate(err)
			return
		}
		switchToBreadcrumb()
	}

	refresh := func(dir string) {
		breadcrumbRow.RemoveAll()
		for _, seg := range pathSegments(dir) {
			target := seg.Path
			breadcrumbRow.Add(widget.NewButton(seg.Name, func() { navigate(b.NavigateTo(target)) }))
		}
		breadcrumbRow.Refresh()
		switchToBreadcrumb()

		if b.CanGoBack() {
			backBtn.Enable()
		} else {
			backBtn.Disable()
		}
		if b.CanGoForward() {
			fwdBtn.Enable()
		} else {
			fwdBtn.Disable()
		}
	}
	b.OnDirChanged = refresh
	refresh(b.CurrentDir())

	buttons := container.NewHBox(backBtn, fwdBtn, upBtn, hiddenCheck)
	pathBar := container.NewStack(breadcrumbTappable, pathEntry)
	return container.NewBorder(nil, nil, buttons, nil, pathBar)
}
