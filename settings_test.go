package gofiledialog

import "testing"

func TestNormalizeSettingsFillsPersistedDefaults(t *testing.T) {
	settings := normalizeSettings(Settings{
		ViewMode:     "bogus",
		SortColumn:   ColumnID(999),
		Columns:      []ColumnSetting{{ID: ColSize, Visible: false}},
		WindowWidth:  -1,
		WindowHeight: 0,
	})

	if settings.ViewMode != string(ViewDetails) {
		t.Fatalf("ViewMode = %q, want %q", settings.ViewMode, ViewDetails)
	}
	if settings.SortColumn != ColName || !settings.SortAscending {
		t.Fatalf("sort = (%v, %v), want (%v, true)", settings.SortColumn, settings.SortAscending, ColName)
	}
	if settings.WindowWidth != 720 || settings.WindowHeight != 480 {
		t.Fatalf("window size = %.0fx%.0f, want 720x480", settings.WindowWidth, settings.WindowHeight)
	}
	if len(settings.Columns) != len(defaultColumns()) {
		t.Fatalf("columns length = %d, want %d", len(settings.Columns), len(defaultColumns()))
	}
	if settings.Columns[0].ID != ColSize {
		t.Fatalf("first persisted column = %v, want %v", settings.Columns[0].ID, ColSize)
	}
}

func TestColumnsFromSettingsRestoresOrderVisibilityAndWidth(t *testing.T) {
	cols := columnsFromSettings([]ColumnSetting{
		{ID: ColSize, Visible: true, Width: 123},
		{ID: ColName, Visible: false, Width: 321},
		{ID: ColType, Visible: false, Width: 44},
	})

	if len(cols) != len(defaultColumns()) {
		t.Fatalf("columns length = %d, want %d", len(cols), len(defaultColumns()))
	}
	if cols[0].ID != ColSize || cols[0].Width != 123 || !cols[0].Visible {
		t.Fatalf("first column = (%v, %.0f, %v), want size/123/visible", cols[0].ID, cols[0].Width, cols[0].Visible)
	}
	if cols[1].ID != ColName || cols[1].Width != 321 || !cols[1].Visible {
		t.Fatalf("name column = (%v, %.0f, %v), want name/321/visible", cols[1].ID, cols[1].Width, cols[1].Visible)
	}
	if cols[2].ID != ColType || cols[2].Width != 44 || cols[2].Visible {
		t.Fatalf("type column = (%v, %.0f, %v), want type/44/hidden", cols[2].ID, cols[2].Width, cols[2].Visible)
	}
}
