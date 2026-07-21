package gofiledialog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListDirTreatsSymlinkToDirectoryAsDirectory(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	entries, err := listDir(base, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name == "link" {
			if !entry.IsDir {
				t.Fatal("symlink to directory was not reported as a directory")
			}
			return
		}
	}
	t.Fatal("symlink entry not found")
}

func TestListDirReturnsReadError(t *testing.T) {
	_, err := listDir(filepath.Join(t.TempDir(), "missing"), true)
	if err == nil {
		t.Fatal("expected missing directory to fail")
	}
}

func TestListDirUsesSymlinkNameForHiddenStatus(t *testing.T) {
	base := t.TempDir()
	visibleTarget := filepath.Join(base, "visible")
	hiddenTarget := filepath.Join(base, ".hidden-target")
	for _, target := range []string{visibleTarget, hiddenTarget} {
		if err := os.WriteFile(target, []byte("file"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(visibleTarget, filepath.Join(base, ".hidden-link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(hiddenTarget, filepath.Join(base, "visible-link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	entries, err := listDir(base, true)
	if err != nil {
		t.Fatal(err)
	}
	hidden := make(map[string]bool, len(entries))
	for _, entry := range entries {
		hidden[entry.Name] = entry.Hidden
	}
	if !hidden[".hidden-link"] {
		t.Fatal("hidden symlink to visible target was not hidden")
	}
	if hidden["visible-link"] {
		t.Fatal("visible symlink to hidden target was hidden")
	}
}
