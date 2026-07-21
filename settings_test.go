package gofiledialog

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNormalizeSettingsFillsPersistedDefaults(t *testing.T) {
	settings := normalizeSettings(Settings{
		ViewMode:     "bogus",
		SortColumn:   ColumnID(999),
		Columns:      []ColumnSetting{{ID: ColSize, Visible: false}},
		WindowWidth:  -1,
		WindowHeight: 0,
	})

	if settings.ViewMode != string(ViewDetails) {
		t.Fatalf("ViewMode = %q, want %q", settings.ViewMode, ViewDetails)
	}
	if settings.SortColumn != ColName || !settings.SortAscending {
		t.Fatalf("sort = (%v, %v), want (%v, true)", settings.SortColumn, settings.SortAscending, ColName)
	}
	if settings.WindowWidth != 720 || settings.WindowHeight != 480 {
		t.Fatalf("window size = %.0fx%.0f, want 720x480", settings.WindowWidth, settings.WindowHeight)
	}
	if len(settings.Columns) != len(defaultColumns()) {
		t.Fatalf("columns length = %d, want %d", len(settings.Columns), len(defaultColumns()))
	}
	if settings.Columns[0].ID != ColSize {
		t.Fatalf("first persisted column = %v, want %v", settings.Columns[0].ID, ColSize)
	}
}

func TestColumnsFromSettingsRestoresOrderVisibilityAndWidth(t *testing.T) {
	cols := columnsFromSettings([]ColumnSetting{
		{ID: ColSize, Visible: true, Width: 123},
		{ID: ColName, Visible: false, Width: 321},
		{ID: ColType, Visible: false, Width: 44},
	})

	if len(cols) != len(defaultColumns()) {
		t.Fatalf("columns length = %d, want %d", len(cols), len(defaultColumns()))
	}
	if cols[0].ID != ColSize || cols[0].Width != 123 || !cols[0].Visible {
		t.Fatalf("first column = (%v, %.0f, %v), want size/123/visible", cols[0].ID, cols[0].Width, cols[0].Visible)
	}
	if cols[1].ID != ColName || cols[1].Width != 321 || !cols[1].Visible {
		t.Fatalf("name column = (%v, %.0f, %v), want name/321/visible", cols[1].ID, cols[1].Width, cols[1].Visible)
	}
	if cols[2].ID != ColType || cols[2].Width != 44 || cols[2].Visible {
		t.Fatalf("type column = (%v, %.0f, %v), want type/44/hidden", cols[2].ID, cols[2].Width, cols[2].Visible)
	}
}

func TestFileStoreConcurrentSavesLeaveValidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store := fileStore{path: path}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			settings := defaultSettings()
			settings.WindowWidth = float32(800 + i)
			settings.WindowHeight = float32(600 + i)
			if err := store.Save(settings); err != nil {
				t.Errorf("Save(%d): %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("settings file is not valid JSON: %v\n%s", err, data)
	}
	if settings.WindowWidth < 800 || settings.WindowWidth > 849 {
		t.Fatalf("WindowWidth = %.0f, want one of the saved values", settings.WindowWidth)
	}
	if matches, err := filepath.Glob(path + ".*.tmp"); err != nil {
		t.Fatal(err)
	} else if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
	unlock, err := acquireSettingsFileLock(path)
	if err != nil {
		t.Fatalf("settings lock remained held after concurrent saves: %v", err)
	}
	unlock()
}

func TestFileStoreRecoversOrphanSettingsLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	lockPath := path + ".lock"
	if err := os.WriteFile(lockPath, []byte("orphan\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := (fileStore{path: path}).Save(defaultSettings()); err != nil {
		t.Fatalf("Save did not recover orphan lock: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("settings were not saved: %v", err)
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("advisory lock file was unexpectedly removed: %v", err)
	}
}

func TestFileStoreDoesNotStealLiveSettingsLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	unlock, err := acquireSettingsFileLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	start := time.Now()
	err = (fileStore{path: path}).Save(defaultSettings())
	if err == nil {
		t.Fatal("Save succeeded while another process-held lock was live")
	}
	if elapsed := time.Since(start); elapsed < settingsLockTimeout {
		t.Fatalf("Save returned after %v, want it to wait for the live lock", elapsed)
	}
}

func TestFileStoreLoadFallsBackForCorruptSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"viewMode":`), 0o644); err != nil {
		t.Fatal(err)
	}

	settings, err := fileStore{path: path}.Load()
	if err != nil {
		t.Fatal(err)
	}
	if settings.ViewMode != string(ViewDetails) || settings.SortColumn != ColName {
		t.Fatalf("settings = %#v, want defaults", settings)
	}
}

func TestDebouncedSaverIgnoresStoreSaveErrors(t *testing.T) {
	store := &failingStore{err: errors.New("disk full")}
	saver := newDebouncedSaver(store, time.Millisecond)

	saver.Flush(defaultSettings())
	saver.Save(defaultSettings())
	time.Sleep(10 * time.Millisecond)

	if got := store.calls(); got < 2 {
		t.Fatalf("Save calls = %d, want at least 2", got)
	}
}

func TestDebouncedSaverSerializesTimerAndFlushSaves(t *testing.T) {
	store := newBlockingStore()
	saver := newDebouncedSaver(store, time.Millisecond)

	timerSettings := defaultSettings()
	timerSettings.WindowWidth = 111
	flushSettings := defaultSettings()
	flushSettings.WindowWidth = 222

	saver.Save(timerSettings)
	select {
	case <-store.firstStarted:
	case <-time.After(time.Second):
		t.Fatal("timer save did not start")
	}

	flushDone := make(chan struct{})
	go func() {
		saver.Flush(flushSettings)
		close(flushDone)
	}()

	time.Sleep(10 * time.Millisecond)
	if max := store.maxActive(); max != 1 {
		t.Fatalf("concurrent Save calls = %d, want 1", max)
	}

	close(store.releaseFirst)
	select {
	case <-flushDone:
	case <-time.After(time.Second):
		t.Fatal("Flush did not complete")
	}

	if max := store.maxActive(); max != 1 {
		t.Fatalf("concurrent Save calls = %d, want 1", max)
	}
	if got := store.lastWidth(); got != 222 {
		t.Fatalf("last saved width = %.0f, want 222", got)
	}
}

type failingStore struct {
	err error
	mu  sync.Mutex
	n   int
}

func (s *failingStore) Load() (Settings, error) {
	return defaultSettings(), nil
}

func (s *failingStore) Save(Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	return s.err
}

func (s *failingStore) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.n
}

type blockingStore struct {
	firstStarted chan struct{}
	releaseFirst chan struct{}

	mu        sync.Mutex
	active    int
	max       int
	calls     int
	lastSaved Settings
}

func newBlockingStore() *blockingStore {
	return &blockingStore{
		firstStarted: make(chan struct{}),
		releaseFirst: make(chan struct{}),
	}
}

func (s *blockingStore) Load() (Settings, error) {
	return defaultSettings(), nil
}

func (s *blockingStore) Save(settings Settings) error {
	s.mu.Lock()
	s.active++
	if s.active > s.max {
		s.max = s.active
	}
	s.calls++
	call := s.calls
	if call == 1 {
		close(s.firstStarted)
	}
	s.mu.Unlock()

	if call == 1 {
		<-s.releaseFirst
	}

	s.mu.Lock()
	s.lastSaved = settings
	s.active--
	s.mu.Unlock()
	return nil
}

func (s *blockingStore) maxActive() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.max
}

func (s *blockingStore) lastWidth() float32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSaved.WindowWidth
}
