package gofiledialog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ColumnSetting is the persisted state of a single Details-view column.
type ColumnSetting struct {
	ID      ColumnID `json:"id"`
	Visible bool     `json:"visible"`
	Width   float32  `json:"width"`
}

// Settings is the dialog state that persists across processes: view mode,
// sort, visible columns and their widths, the hidden-files toggle, and
// window size. It is shared by every app that imports gofiledialog and uses
// the default Store (see WithStore to opt out).
type Settings struct {
	ViewMode      string          `json:"viewMode"`
	ShowHidden    bool            `json:"showHidden"`
	SortColumn    ColumnID        `json:"sortColumn"`
	SortAscending bool            `json:"sortAscending"`
	Columns       []ColumnSetting `json:"columns"`
	WindowWidth   float32         `json:"windowWidth"`
	WindowHeight  float32         `json:"windowHeight"`
}

func defaultSettings() Settings {
	cols := defaultColumns()
	colSettings := make([]ColumnSetting, len(cols))
	for i, c := range cols {
		colSettings[i] = ColumnSetting{ID: c.ID, Visible: c.Visible, Width: c.Width}
	}
	return Settings{
		ViewMode:      "details",
		ShowHidden:    false,
		SortColumn:    ColName,
		SortAscending: true,
		Columns:       colSettings,
		WindowWidth:   720,
		WindowHeight:  480,
	}
}

func normalizeSettings(s Settings) Settings {
	defaults := defaultSettings()

	if viewModeFromString(s.ViewMode) == ViewDetails && s.ViewMode != string(ViewDetails) {
		s.ViewMode = defaults.ViewMode
	}
	if !knownColumnID(s.SortColumn) {
		s.SortColumn = defaults.SortColumn
		s.SortAscending = defaults.SortAscending
	}
	if len(s.Columns) == 0 {
		s.Columns = defaults.Columns
	} else {
		s.Columns = normalizeColumnSettings(s.Columns)
	}
	if s.WindowWidth <= 0 {
		s.WindowWidth = defaults.WindowWidth
	}
	if s.WindowHeight <= 0 {
		s.WindowHeight = defaults.WindowHeight
	}
	return s
}

func normalizeColumnSettings(settings []ColumnSetting) []ColumnSetting {
	defaults := defaultSettings().Columns
	known := make(map[ColumnID]ColumnSetting, len(defaults))
	for _, cs := range defaults {
		known[cs.ID] = cs
	}

	seen := make(map[ColumnID]bool, len(defaults))
	out := make([]ColumnSetting, 0, len(defaults))
	for _, cs := range settings {
		base, ok := known[cs.ID]
		if !ok || seen[cs.ID] {
			continue
		}
		seen[cs.ID] = true
		if cs.ID == ColName {
			cs.Visible = true
		}
		if cs.Width <= 0 {
			cs.Width = base.Width
		}
		out = append(out, cs)
	}
	for _, cs := range defaults {
		if seen[cs.ID] {
			continue
		}
		out = append(out, cs)
	}
	return out
}

// Store loads and saves Settings. The default, used unless an app supplies
// its own via WithStore, is a JSON file shared across every app on the
// machine that uses gofiledialog with its default settings.
type Store interface {
	Load() (Settings, error)
	Save(Settings) error
}

// defaultSettingsPath returns the shared settings file path, or an error if
// the OS config directory can't be determined.
func defaultSettingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "wierdling-gofiledialog", "settings.json"), nil
}

type fileStore struct {
	path string
}

// newFileStore resolves the default Store. If the OS config directory can't
// be determined, it falls back to a no-op store (settings simply won't
// persist) rather than failing dialog construction.
func newFileStore() Store {
	path, err := defaultSettingsPath()
	if err != nil {
		return noopStore{}
	}
	return fileStore{path: path}
}

// Load returns the saved settings, or defaults if the file is missing or
// unreadable/corrupt — persistence failures should never break the dialog.
func (s fileStore) Load() (Settings, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return defaultSettings(), nil
	}
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaultSettings(), nil
	}
	return normalizeSettings(settings), nil
}

func (s fileStore) Save(settings Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

type noopStore struct{}

func (noopStore) Load() (Settings, error) { return defaultSettings(), nil }
func (noopStore) Save(Settings) error     { return nil }

// debouncedSaver coalesces rapid successive setting changes (e.g. repeated
// column toggles) into a single write, plus a Flush for saving immediately
// on dialog close.
type debouncedSaver struct {
	store Store
	delay time.Duration

	mu    sync.Mutex
	timer *time.Timer
}

func newDebouncedSaver(store Store, delay time.Duration) *debouncedSaver {
	return &debouncedSaver{store: store, delay: delay}
}

func (d *debouncedSaver) Save(s Settings) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.delay, func() { d.store.Save(s) })
}

func (d *debouncedSaver) Flush(s Settings) {
	d.mu.Lock()
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	d.mu.Unlock()
	d.store.Save(s)
}
