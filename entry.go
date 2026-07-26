package gofiledialog

import (
	"os"
	"path/filepath"
	"time"
)

// FileEntry describes a single file or directory shown in the browser.
type FileEntry struct {
	Name        string
	Path        string
	IsDir       bool
	Hidden      bool
	Size        int64
	ModTime     time.Time
	CreatedTime time.Time
	// Mode preserves the filesystem type so callers can distinguish regular
	// files from special entries (sockets, devices, and named pipes).
	Mode os.FileMode
}

func newFileEntry(dir string, info os.FileInfo) FileEntry {
	return FileEntry{
		Name:        info.Name(),
		Path:        filepath.Join(dir, info.Name()),
		IsDir:       info.IsDir(),
		Hidden:      isHidden(info),
		Size:        info.Size(),
		ModTime:     info.ModTime(),
		CreatedTime: createdTime(info),
		Mode:        info.Mode(),
	}
}
