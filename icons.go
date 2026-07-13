package gofiledialog

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// entryIconResource returns the generic file-type icon for entry. Image
// files additionally get a real thumbnail requested asynchronously by the
// icon-view widgets; this is just the immediate placeholder/fallback.
func entryIconResource(entry FileEntry) fyne.Resource {
	if entry.IsDir {
		return theme.FolderIcon()
	}
	switch strings.ToLower(filepath.Ext(entry.Name)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg", ".webp", ".tiff":
		return theme.FileImageIcon()
	case ".mp3", ".wav", ".flac", ".aac", ".ogg", ".m4a", ".wma":
		return theme.FileAudioIcon()
	case ".mp4", ".mov", ".avi", ".mkv", ".webm", ".wmv":
		return theme.FileVideoIcon()
	case ".zip", ".tar", ".gz", ".rar", ".7z", ".exe", ".msi", ".dll", ".app":
		return theme.FileApplicationIcon()
	case ".txt", ".md", ".go", ".c", ".cpp", ".h", ".py", ".js", ".ts", ".java",
		".rs", ".json", ".xml", ".yaml", ".yml", ".html", ".css", ".sh":
		return theme.FileTextIcon()
	default:
		return theme.FileIcon()
	}
}
