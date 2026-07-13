package gofiledialog

import "fyne.io/fyne/v2/widget"

// ViewMode selects how the Browser displays its entries.
type ViewMode string

const (
	ViewDetails     ViewMode = "details"
	ViewList        ViewMode = "list"
	ViewSmallIcons  ViewMode = "small"
	ViewMediumIcons ViewMode = "medium"
	ViewLargeIcons  ViewMode = "large"
)

// allViewModes lists every mode in the order they should appear in a view
// switcher.
var allViewModes = []ViewMode{ViewDetails, ViewList, ViewSmallIcons, ViewMediumIcons, ViewLargeIcons}

// Label returns the human-readable name for m, e.g. for a view-switcher menu.
func (m ViewMode) Label() string {
	switch m {
	case ViewDetails:
		return "Details"
	case ViewList:
		return "List"
	case ViewSmallIcons:
		return "Small icons"
	case ViewMediumIcons:
		return "Medium icons"
	case ViewLargeIcons:
		return "Large icons"
	default:
		return string(m)
	}
}

func viewModeFromLabel(label string) ViewMode {
	for _, m := range allViewModes {
		if m.Label() == label {
			return m
		}
	}
	return ViewDetails
}

// viewModeFromString validates a persisted view-mode string, falling back
// to Details for anything unrecognized (e.g. an older/corrupt settings file).
func viewModeFromString(s string) ViewMode {
	for _, m := range allViewModes {
		if string(m) == s {
			return m
		}
	}
	return ViewDetails
}

// ViewMode returns the currently active view.
func (b *Browser) ViewMode() ViewMode {
	return b.viewMode
}

// SetViewMode switches the active view.
func (b *Browser) SetViewMode(mode ViewMode) {
	mode = viewModeFromString(string(mode))
	if mode == b.viewMode {
		return
	}
	b.viewMode = mode
	b.showActiveView()
	b.settingsChanged()
}

// newViewSwitcher builds a "View" dropdown that switches b's active view.
// onChange, if non-nil, fires after the switch (e.g. to enable/disable the
// Columns menu, which only applies to the Details view).
func newViewSwitcher(b *Browser, onChange func(ViewMode)) *widget.Select {
	labels := make([]string, len(allViewModes))
	for i, m := range allViewModes {
		labels[i] = m.Label()
	}
	sel := widget.NewSelect(labels, func(label string) {
		mode := viewModeFromLabel(label)
		b.SetViewMode(mode)
		if onChange != nil {
			onChange(mode)
		}
	})
	sel.SetSelected(b.ViewMode().Label())
	return sel
}

func (b *Browser) showActiveView() {
	for _, obj := range b.stack.Objects {
		obj.Hide()
	}
	switch b.viewMode {
	case ViewDetails:
		b.table.Show()
	case ViewList:
		b.list.Show()
	default:
		if grid, ok := b.grids[b.viewMode]; ok {
			grid.Show()
		} else {
			b.table.Show()
		}
	}
	b.stack.Refresh()
}
