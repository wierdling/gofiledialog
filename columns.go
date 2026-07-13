package gofiledialog

import "strings"

// ColumnID identifies one of the built-in Details-view columns.
type ColumnID int

const (
	ColName ColumnID = iota
	ColDateModified
	ColType
	ColSize
	ColDateCreated
)

func knownColumnID(id ColumnID) bool {
	for _, col := range defaultColumns() {
		if col.ID == id {
			return true
		}
	}
	return false
}

// Column describes one Details-view column: how to render a cell and how to
// compare two entries for sorting.
type Column struct {
	ID      ColumnID
	Title   string
	Width   float32
	Visible bool
	Text    func(e FileEntry) string
	Less    func(a, b FileEntry) bool
}

// defaultColumns returns the built-in column set in default display order.
// Name is always visible; callers must not allow it to be hidden.
func defaultColumns() []Column {
	return []Column{
		{
			ID: ColName, Title: "Name", Width: 240, Visible: true,
			Text: func(e FileEntry) string { return e.Name },
			Less: func(a, b FileEntry) bool { return strings.ToLower(a.Name) < strings.ToLower(b.Name) },
		},
		{
			ID: ColDateModified, Title: "Date modified", Width: 150, Visible: true,
			Text: formatModTime,
			Less: func(a, b FileEntry) bool { return a.ModTime.Before(b.ModTime) },
		},
		{
			ID: ColType, Title: "Type", Width: 100, Visible: true,
			Text: formatType,
			Less: func(a, b FileEntry) bool { return formatType(a) < formatType(b) },
		},
		{
			ID: ColSize, Title: "Size", Width: 90, Visible: true,
			Text: func(e FileEntry) string { return formatSize(e.IsDir, e.Size) },
			Less: func(a, b FileEntry) bool { return a.Size < b.Size },
		},
		{
			ID: ColDateCreated, Title: "Date created", Width: 150, Visible: supportsCreatedTime,
			Text: formatCreatedTime,
			Less: func(a, b FileEntry) bool { return a.CreatedTime.Before(b.CreatedTime) },
		},
	}
}
