package gofiledialog

import (
	"path/filepath"
	"strings"
)

// Filter describes one file-type option in the dialog's Type dropdown.
// Extensions are case-insensitive and may be written with or without a
// leading dot. A nil or empty Extensions slice matches all files.
type Filter struct {
	Name       string
	Extensions []string
}

func (f Filter) Label() string {
	if f.Name != "" {
		return f.Name
	}
	if len(f.Extensions) == 0 {
		return "All files"
	}
	return strings.Join(normalizeExtensions(f.Extensions), ", ")
}

func (f Filter) Match(entry FileEntry) bool {
	if entry.IsDir || len(f.Extensions) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(entry.Name))
	for _, allowed := range normalizeExtensions(f.Extensions) {
		if ext == allowed {
			return true
		}
	}
	return false
}

func (f Filter) DefaultExtension() string {
	for _, ext := range normalizeExtensions(f.Extensions) {
		if ext != "" {
			return ext
		}
	}
	return ""
}

func normalizeExtensions(exts []string) []string {
	out := make([]string, 0, len(exts))
	seen := make(map[string]bool, len(exts))
	for _, ext := range exts {
		ext = strings.TrimSpace(strings.ToLower(ext))
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		if seen[ext] {
			continue
		}
		seen[ext] = true
		out = append(out, ext)
	}
	return out
}

func ensureExtension(name string, filter Filter) string {
	if filepath.Ext(name) != "" {
		return name
	}
	if ext := filter.DefaultExtension(); ext != "" {
		return name + ext
	}
	return name
}
