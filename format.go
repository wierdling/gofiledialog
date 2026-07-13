package gofiledialog

import (
	"fmt"
	"path/filepath"
	"strings"
)

func formatSize(dir bool, size int64) string {
	if dir {
		return ""
	}
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.0f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func formatType(e FileEntry) string {
	if e.IsDir {
		return "File folder"
	}
	ext := filepath.Ext(e.Name)
	if ext == "" {
		return "File"
	}
	return strings.ToUpper(strings.TrimPrefix(ext, ".")) + " File"
}

func formatModTime(e FileEntry) string {
	return e.ModTime.Format("1/2/2006 3:04 PM")
}

func formatCreatedTime(e FileEntry) string {
	if e.CreatedTime.IsZero() {
		return "—"
	}
	return e.CreatedTime.Format("1/2/2006 3:04 PM")
}
