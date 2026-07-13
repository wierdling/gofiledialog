package gofiledialog

import (
	"path/filepath"
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
