package gofiledialog

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

type dialogCallbackResult struct {
	paths []string
	err   error
}

type memoryStore struct {
	settings Settings
}

func (s *memoryStore) Load() (Settings, error) { return s.settings, nil }
func (s *memoryStore) Save(settings Settings) error {
	s.settings = settings
	return nil
}

func TestStartDirPrefersFyneLastFolder(t *testing.T) {
	app := fynetest.NewApp()
	fyneDir := t.TempDir()
	app.Preferences().SetString(fyneLastFolderKey, storage.NewFileURI(fyneDir).String())

	if got := startDir("", Settings{LastDir: t.TempDir()}); got != fyneDir {
		t.Fatalf("startDir() = %q, want Fyne last folder %q", got, fyneDir)
	}
}

func TestStartDirFallsBackToDialogSettings(t *testing.T) {
	fynetest.NewApp()
	want := t.TempDir()

	if got := startDir("", Settings{LastDir: want}); got != want {
		t.Fatalf("startDir() = %q, want persisted dialog folder %q", got, want)
	}
}

func TestDialogClosePersistsLastDirectory(t *testing.T) {
	fynetest.NewApp()
	store := &memoryStore{}
	want := t.TempDir()
	d, err := NewOpen(nil, WithStartDir(want), WithStore(store))
	if err != nil {
		t.Fatal(err)
	}
	d.win.Close()

	if store.settings.LastDir != want {
		t.Fatalf("saved LastDir = %q, want %q", store.settings.LastDir, want)
	}
}

func TestOpenDialogWindowCloseCancels(t *testing.T) {
	fynetest.NewApp()
	d := newTestOpenDialog(t)
	var got []dialogCallbackResult
	d.SetOnChosen(func(paths []string, err error) {
		got = append(got, dialogCallbackResult{paths: paths, err: err})
	})

	d.win.Close()

	assertOneCallback(t, got, nil, nil)
}

func TestOpenDialogCancelIsExactlyOnce(t *testing.T) {
	fynetest.NewApp()
	d := newTestOpenDialog(t)
	var got []dialogCallbackResult
	d.SetOnChosen(func(paths []string, err error) {
		got = append(got, dialogCallbackResult{paths: paths, err: err})
	})

	d.cancel()
	d.cancel()

	assertOneCallback(t, got, nil, nil)
}

func TestOpenDialogSelectionBeatsWindowCloseCancellation(t *testing.T) {
	fynetest.NewApp()
	d := newTestOpenDialog(t)
	path := filepath.Join(d.browser.CurrentDir(), "file.txt")
	if err := os.WriteFile(path, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	d.browser.applyListing(d.browser.CurrentDir(), []FileEntry{{Name: "file.txt", Path: path}})
	d.browser.selectRow(0)
	var got []dialogCallbackResult
	d.SetOnChosen(func(paths []string, err error) {
		got = append(got, dialogCallbackResult{paths: paths, err: err})
	})

	d.chooseSelected()
	d.cancel()

	assertOneCallback(t, got, []string{path}, nil)
}

func TestOpenDialogChoosesTypedFile(t *testing.T) {
	fynetest.NewApp()
	d := newTestOpenDialog(t)
	path := filepath.Join(d.browser.CurrentDir(), "typed.txt")
	if err := os.WriteFile(path, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	d.fileEntry.SetText("typed.txt")
	var got []dialogCallbackResult
	d.SetOnChosen(func(paths []string, err error) { got = append(got, dialogCallbackResult{paths: paths, err: err}) })

	d.chooseSelected()

	assertOneCallback(t, got, []string{path}, nil)
}

func TestOpenDialogDoesNotChooseMissingTypedFile(t *testing.T) {
	fynetest.NewApp()
	d := newTestOpenDialog(t)
	d.fileEntry.SetText("missing.txt")
	var got []dialogCallbackResult
	d.SetOnChosen(func(paths []string, err error) { got = append(got, dialogCallbackResult{paths: paths, err: err}) })

	d.chooseSelected()

	if len(got) != 0 {
		t.Fatalf("callback count = %d, want 0", len(got))
	}
}

func TestOpenDialogRejectsMultipleTypedFilesWithoutMultiSelect(t *testing.T) {
	fynetest.NewApp()
	d := newTestOpenDialog(t)
	for _, name := range []string{"one.txt", "two.txt"} {
		if err := os.WriteFile(filepath.Join(d.browser.CurrentDir(), name), []byte("file"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	d.fileEntry.SetText("one.txt; two.txt")
	var got []dialogCallbackResult
	d.SetOnChosen(func(paths []string, err error) { got = append(got, dialogCallbackResult{paths: paths, err: err}) })

	d.chooseSelected()

	if len(got) != 0 {
		t.Fatalf("callback count = %d, want 0", len(got))
	}
}

func TestSaveDialogWindowCloseCancels(t *testing.T) {
	fynetest.NewApp()
	d := newTestSaveDialog(t)
	var got []dialogCallbackResult
	d.SetOnChosen(func(paths []string, err error) {
		got = append(got, dialogCallbackResult{paths: paths, err: err})
	})

	d.win.Close()

	assertOneCallback(t, got, nil, nil)
}

func TestSaveDialogOverwriteConfirmCannotCompleteAfterCancel(t *testing.T) {
	fynetest.NewApp()
	d := newTestSaveDialog(t)
	path := filepath.Join(d.browser.CurrentDir(), "existing.txt")
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	d.fileEntry.SetText(filepath.Base(path))
	var got []dialogCallbackResult
	d.SetOnChosen(func(paths []string, err error) {
		got = append(got, dialogCallbackResult{paths: paths, err: err})
	})

	d.chooseSave()
	d.cancel()
	confirm := findButtonByText(d.win.Canvas().Overlays().Top(), "Yes")
	if confirm == nil {
		t.Fatal("expected overwrite confirmation button")
	}
	fynetest.Tap(confirm)

	assertOneCallback(t, got, nil, nil)
}

func TestSaveDialogDoesNotOverwriteDirectory(t *testing.T) {
	fynetest.NewApp()
	d := newTestSaveDialog(t)
	directory := filepath.Join(d.browser.CurrentDir(), "existing")
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	d.fileEntry.SetText("existing")
	var got []dialogCallbackResult
	d.SetOnChosen(func(paths []string, err error) { got = append(got, dialogCallbackResult{paths: paths, err: err}) })

	d.chooseSave()

	if len(got) != 0 {
		t.Fatalf("callback count = %d, want 0", len(got))
	}
	if confirm := findButtonByText(d.win.Canvas().Overlays().Top(), "Yes"); confirm != nil {
		t.Fatal("directory target displayed an overwrite confirmation")
	}
}

func TestFolderDialogWindowCloseCancels(t *testing.T) {
	fynetest.NewApp()
	d := newTestFolderDialog(t)
	var got []dialogCallbackResult
	d.SetOnChosen(func(paths []string, err error) {
		got = append(got, dialogCallbackResult{paths: paths, err: err})
	})

	d.win.Close()

	assertOneCallback(t, got, nil, nil)
}

func newTestOpenDialog(t *testing.T) *OpenDialog {
	t.Helper()
	d, err := NewOpen(nil, WithStartDir(t.TempDir()), WithStore(noopStore{}))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func newTestSaveDialog(t *testing.T) *SaveDialog {
	t.Helper()
	d, err := NewSave(nil, WithStartDir(t.TempDir()), WithStore(noopStore{}))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func newTestFolderDialog(t *testing.T) *FolderDialog {
	t.Helper()
	d, err := NewFolder(nil, WithStartDir(t.TempDir()), WithStore(noopStore{}))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func assertOneCallback(t *testing.T, got []dialogCallbackResult, wantPaths []string, wantErr error) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("callback count = %d, want 1", len(got))
	}
	if !sameStringSlice(got[0].paths, wantPaths) {
		t.Fatalf("paths = %#v, want %#v", got[0].paths, wantPaths)
	}
	if !errors.Is(got[0].err, wantErr) {
		t.Fatalf("err = %v, want %v", got[0].err, wantErr)
	}
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func findButtonByText(obj fyne.CanvasObject, text string) *widget.Button {
	switch o := obj.(type) {
	case nil:
		return nil
	case *widget.Button:
		if o.Text == text {
			return o
		}
	case *widget.PopUp:
		return findButtonByText(o.Content, text)
	case *fyne.Container:
		for _, child := range o.Objects {
			if button := findButtonByText(child, text); button != nil {
				return button
			}
		}
	}
	return nil
}
