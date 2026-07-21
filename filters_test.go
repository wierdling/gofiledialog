package gofiledialog

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFilterMatchUsesCaseInsensitiveExtensions(t *testing.T) {
	filter := Filter{Name: "Images", Extensions: []string{"png", ".JPG"}}

	if !filter.Match(FileEntry{Name: "photo.PNG"}) {
		t.Fatal("expected .PNG to match png filter")
	}
	if !filter.Match(FileEntry{Name: "folder", IsDir: true}) {
		t.Fatal("directories should always match filters")
	}
	if filter.Match(FileEntry{Name: "notes.txt"}) {
		t.Fatal("txt should not match image filter")
	}
}

func TestSavePathAddsDefaultExtension(t *testing.T) {
	got, err := savePath(filepath.Join("C:", "tmp"), "report", Filter{Extensions: []string{"txt"}})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "report.txt" {
		t.Fatalf("save path base = %q, want report.txt", filepath.Base(got))
	}

	got, err = savePath(filepath.Join("C:", "tmp"), "report.md", Filter{Extensions: []string{"txt"}})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "report.md" {
		t.Fatalf("save path base = %q, want report.md", filepath.Base(got))
	}
}

func TestSavePathRejectsBlankName(t *testing.T) {
	if _, err := savePath(".", "   ", Filter{}); err == nil {
		t.Fatal("expected blank filename to fail")
	}
}

func TestValidateFilterLabelsRejectsDuplicateNames(t *testing.T) {
	err := validateFilterLabels([]Filter{
		{Name: "Images", Extensions: []string{".png"}},
		{Name: "Images", Extensions: []string{".jpg"}},
	})
	if err == nil {
		t.Fatal("expected duplicate named filters to fail")
	}
	if !strings.Contains(err.Error(), `"Images"`) {
		t.Fatalf("error = %q, want duplicate label", err)
	}
}

func TestValidateFilterLabelsRejectsDuplicateGeneratedLabels(t *testing.T) {
	err := validateFilterLabels([]Filter{
		{Extensions: []string{"png", ".jpg"}},
		{Extensions: []string{".png", "jpg"}},
	})
	if err == nil {
		t.Fatal("expected duplicate generated labels to fail")
	}
	if !strings.Contains(err.Error(), `".png, .jpg"`) {
		t.Fatalf("error = %q, want duplicate generated label", err)
	}
}

func TestValidateFilterLabelsAllowsUniqueLabels(t *testing.T) {
	if err := validateFilterLabels([]Filter{
		{Name: "Images", Extensions: []string{".png"}},
		{Name: "Documents", Extensions: []string{".pdf"}},
		{Extensions: []string{".txt"}},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDialogConstructorsRejectDuplicateFilterLabels(t *testing.T) {
	_, err := NewOpen(nil,
		WithStartDir(t.TempDir()),
		WithStore(noopStore{}),
		WithFilters(
			Filter{Name: "Images", Extensions: []string{".png"}},
			Filter{Name: "Images", Extensions: []string{".jpg"}},
		),
	)
	if err == nil {
		t.Fatal("expected NewOpen to reject duplicate filter labels")
	}

	_, err = NewSave(nil,
		WithStartDir(t.TempDir()),
		WithStore(noopStore{}),
		WithFilters(
			Filter{Extensions: []string{".txt"}},
			Filter{Extensions: []string{"txt"}},
		),
	)
	if err == nil {
		t.Fatal("expected NewSave to reject duplicate filter labels")
	}
}
