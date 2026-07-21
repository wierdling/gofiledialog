package gofiledialog

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewFolderPathRestrictsInputToSingleFolderName(t *testing.T) {
	base := t.TempDir()
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "", wantErr: true},
		{name: "   ", wantErr: true},
		{name: " leading", wantErr: true},
		{name: "trailing ", wantErr: true},
		{name: ".", wantErr: true},
		{name: "..", wantErr: true},
		{name: `..\outside`, wantErr: true},
		{name: "../outside", wantErr: true},
		{name: `nested\child`, wantErr: true},
		{name: "nested/child", wantErr: true},
		{name: string([]byte{'b', 'a', 'd', 0}), wantErr: true},
		{name: filepath.Join(base, "absolute"), wantErr: true},
		{name: "Photos 2026"},
		{name: "新しいフォルダー"},
	}

	for _, tt := range tests {
		t.Run(printableTestName(tt.name), func(t *testing.T) {
			got, err := newFolderPath(base, tt.name)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("newFolderPath(%q) succeeded with %q, want error", tt.name, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("newFolderPath(%q): %v", tt.name, err)
			}
			want := filepath.Join(base, tt.name)
			if got != want {
				t.Fatalf("path = %q, want %q", got, want)
			}
			if !pathInsideDir(t, base, got) {
				t.Fatalf("path %q is outside base dir %q", got, base)
			}
		})
	}
}

func TestNewFolderPathRejectsWindowsInvalidNames(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific folder name rules")
	}

	for _, name := range []string{"bad:name", "bad?", "CON", "con.txt", "folder.", "folder "} {
		t.Run(name, func(t *testing.T) {
			if got, err := newFolderPath(t.TempDir(), name); err == nil {
				t.Fatalf("newFolderPath(%q) succeeded with %q, want error", name, got)
			}
		})
	}
}

func TestNewFolderPathAllowsExistingDirectoryErrorToComeFromMkdir(t *testing.T) {
	base := t.TempDir()
	path, err := newFolderPath(base, "already-there")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); !os.IsExist(err) {
		t.Fatalf("second Mkdir error = %v, want already-exists error", err)
	}
}

func pathInsideDir(t *testing.T, dir, path string) bool {
	t.Helper()
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func printableTestName(name string) string {
	if name == "" {
		return "empty"
	}
	return strings.Map(func(r rune) rune {
		if r == 0 {
			return -1
		}
		return r
	}, name)
}
