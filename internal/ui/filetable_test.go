package ui

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/maarten-boot/fileViews/internal/contrast"
	"github.com/maarten-boot/fileViews/internal/listing"
)

// cellText renders one table cell and returns its text and alignment.
func cellText(ft *fileTable, row, col int) (string, fyne.TextAlign) {
	c := newFileCell(func(int) {}, func(int) {})
	ft.updateCell(widget.TableCellID{Row: row, Col: col}, c)
	return c.label.Text, c.label.Alignment
}

func rowTexts(ft *fileTable, row int) []string {
	var out []string
	for col := range ft.cols {
		text, _ := cellText(ft, row, col)
		out = append(out, text)
	}
	return out
}

// stubOpen replaces openFile for the test and records what was opened.
func stubOpen(t *testing.T) *[]string {
	t.Helper()
	var opened []string
	old := openFile
	openFile = func(p string) error { opened = append(opened, p); return nil }
	t.Cleanup(func() { openFile = old })
	return &opened
}

// TC-022c, TC-023f: cells show the ls -lF columns in the default order.
func TestTableCells(t *testing.T) {
	root := fixture(t)
	os.Chmod(filepath.Join(root, "a.txt"), 0o640)
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) })

	if len(m.files.cols) != 7 {
		t.Fatalf("columns = %v", m.files.cols)
	}
	row := rowTexts(m.files, 1) // a.txt
	if row[0] != "-rw-r-----" || row[1] != "1" || row[4] != "100" || row[6] != "a.txt" {
		t.Errorf("row = %v", row)
	}
	if row[2] == "" || row[3] == "" || len(row[5]) != len("2006-01-02 15:04") {
		t.Errorf("owner/group/modified = %v", row[2:6])
	}
	if text, _ := cellText(m.files, 0, 6); text != "sub/" {
		t.Errorf("dir name = %q", text)
	}
	for col, want := range map[int]fyne.TextAlign{1: fyne.TextAlignTrailing, 4: fyne.TextAlignTrailing, 6: fyne.TextAlignLeading} {
		if _, got := cellText(m.files, 1, col); got != want {
			t.Errorf("column %d alignment %v, want %v", col, got, want)
		}
	}
	// Exact sizes switch the Size column too.
	os.WriteFile(filepath.Join(root, "big"), make([]byte, 4300), 0o644)
	wait(t, m, m.refresh)
	if text, _ := cellText(m.files, 2, 4); text != "4.2K" {
		t.Errorf("human size = %q", text)
	}
	m.toggleExactSizes()
	if text, _ := cellText(m.files, 2, 4); text != "4300" {
		t.Errorf("exact size = %q", text)
	}
}

// TC-024e: the table follows the column layout.
func TestTableColumns(t *testing.T) {
	m := newTestWindow(t, t.TempDir())
	m.files.setColumns([]listing.Column{listing.ColName, listing.ColSize})
	rows, cols := m.files.table.Length()
	if rows != 0 || cols != 2 {
		t.Errorf("Length() = %d, %d", rows, cols)
	}
	h := newHeaderCell(func(int) {})
	m.files.updateHeader(widget.TableCellID{Row: -1, Col: 1}, h)
	if h.label.Text != "Size" {
		t.Errorf("header = %q", h.label.Text)
	}
}

// TC-025e: clicking headers sorts, a second click reverses, dirs stay first.
func TestTableSortByHeader(t *testing.T) {
	root := fixture(t)
	os.Mkdir(filepath.Join(root, "zdir"), 0o755)
	os.WriteFile(filepath.Join(root, "b.bin"), make([]byte, 999), 0o644)
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) })
	if got := shownNames(m); !slices.Equal(got, []string{"sub/", "zdir/", "a.txt", "b.bin"}) {
		t.Fatalf("default order %v", got)
	}

	header := func(col listing.Column) *headerCell {
		h := newHeaderCell(m.files.sortBy)
		m.files.updateHeader(widget.TableCellID{Row: -1, Col: slices.Index(m.files.cols, col)}, h)
		return h
	}
	if header(listing.ColName).icon.Resource != theme.MoveUpIcon() || header(listing.ColSize).icon.Resource != nil {
		t.Error("default header icons wrong")
	}

	test.Tap(header(listing.ColSize))
	if got := shownNames(m); !slices.Equal(got[2:], []string{"a.txt", "b.bin"}) || m.cfg.SortSpec() != (listing.SortSpec{Column: listing.ColSize}) {
		t.Errorf("size ascending: %v %+v", got, m.cfg.SortSpec())
	}
	test.Tap(header(listing.ColSize))
	if got := shownNames(m); !slices.Equal(got, []string{"zdir/", "sub/", "b.bin", "a.txt"}) { // equal dir sizes: name order reversed too
		t.Errorf("size descending: %v", got)
	}
	if header(listing.ColSize).icon.Resource != theme.MoveDownIcon() || header(listing.ColName).icon.Resource != nil {
		t.Error("header icons after sorting by size")
	}
	test.Tap(header(listing.ColName))
	if m.cfg.SortSpec() != listing.DefaultSort {
		t.Errorf("back to name: %+v", m.cfg.SortSpec())
	}
}

// TC-020b: whole-row selection that follows the entry.
func TestTableSelection(t *testing.T) {
	root := fixture(t)
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) })

	var tapped []int
	c := newFileCell(func(r int) { tapped = append(tapped, r); m.clickRow(r, 0) }, func(int) {})
	m.files.updateCell(widget.TableCellID{Row: 1, Col: 3}, c)
	test.Tap(c)
	if !slices.Equal(tapped, []int{1}) || !m.sel.has("a.txt") {
		t.Fatalf("tapped %v selection %v", tapped, m.sel.names)
	}
	// Every cell of the selected row is highlighted, other rows are not.
	for col := range m.files.cols {
		m.files.updateCell(widget.TableCellID{Row: 1, Col: col}, c)
		if c.bg.FillColor != theme.Color(theme.ColorNameSelection) {
			t.Errorf("selected row, column %d not highlighted", col)
		}
	}
	m.files.updateCell(widget.TableCellID{Row: 0, Col: 0}, c)
	if c.bg.FillColor != nil {
		t.Error("unselected row highlighted")
	}

	// Selection follows the entry when sorting and filtering.
	m.files.sortBy(int(listing.ColName)) // name descending
	if names := entryNames(m.sel.entries(m.shown)); !slices.Equal(names, []string{"a.txt"}) {
		t.Errorf("after sort: %v", names)
	}
	m.toggleHidden()
	if !m.sel.has("a.txt") {
		t.Error("lost selection when showing hidden files")
	}
	// Refresh keeps it; going elsewhere clears it.
	wait(t, m, m.refresh)
	if !m.sel.has("a.txt") {
		t.Error("refresh cleared the selection")
	}
	wait(t, m, func() { m.goPath(filepath.Join(root, "sub")) })
	if len(m.sel.names) != 0 {
		t.Errorf("navigation kept selection %v", m.sel.names)
	}
}

// TC-020c, TC-050f: Ctrl/Shift+click in the window; status shows count and size.
func TestTableMultiSelect(t *testing.T) {
	root := fixture(t)
	os.WriteFile(filepath.Join(root, "b.txt"), make([]byte, 200), 0o644)
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) }) // sub/, a.txt, b.txt
	m.clickRow(1, 0)
	m.clickRow(2, fyne.KeyModifierControl)
	if !strings.Contains(m.status.Text, "2 selected (300)") {
		t.Errorf("status = %q", m.status.Text)
	}
	m.clickRow(0, fyne.KeyModifierShift) // range from b.txt up to sub/
	if !strings.Contains(m.status.Text, "3 selected (300)") {
		t.Errorf("status = %q", m.status.Text)
	}
	// Hiding a selected hidden entry drops it from the selection.
	m.toggleHidden()
	m.clickRow(slices.Index(entryNames(m.shown), ".hidden"), fyne.KeyModifierControl)
	m.toggleHidden()
	if m.sel.has(".hidden") || !strings.Contains(m.status.Text, "3 selected") {
		t.Errorf("hidden entry still selected: %v, %q", m.sel.names, m.status.Text)
	}
}

// TC-027: double-clicking a directory or a link to one opens it.
func TestTableOpenDirectory(t *testing.T) {
	root := fixture(t)
	os.Symlink("sub", filepath.Join(root, "link"))
	opened := stubOpen(t)
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) })

	c := newFileCell(func(int) {}, m.files.openRow)
	m.files.updateCell(widget.TableCellID{Row: slices.Index(shownNames(m), "sub/"), Col: 0}, c)
	wait(t, m, func() { test.DoubleTap(c) })
	if m.nav.Current() != filepath.Join(root, "sub") {
		t.Errorf("current = %q", m.nav.Current())
	}
	wait(t, m, m.goBack)
	wait(t, m, func() { m.files.openRow(slices.Index(shownNames(m), "link@ -> sub")) })
	if m.nav.Current() != filepath.Join(root, "link") {
		t.Errorf("via link: current = %q", m.nav.Current())
	}
	if len(*opened) != 0 {
		t.Errorf("directory opened as file: %v", *opened)
	}
}

// TC-028: double-clicking a regular file opens it with the default app;
// other kinds of entries are refused.
func TestTableOpenFile(t *testing.T) {
	root := fixture(t)
	if err := syscall.Mkfifo(filepath.Join(root, "pipe"), 0o644); err != nil {
		t.Fatal(err)
	}
	opened := stubOpen(t)
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) })

	m.files.openRow(slices.Index(shownNames(m), "a.txt"))
	if want := []string{filepath.Join(root, "a.txt")}; !slices.Equal(*opened, want) {
		t.Errorf("opened %v, want %v", *opened, want)
	}
	m.files.openRow(slices.Index(shownNames(m), "pipe|"))
	if len(*opened) != 1 {
		t.Errorf("FIFO was opened: %v", *opened)
	}
	// An entry removed since the listing: error, nothing opened.
	os.Remove(filepath.Join(root, "a.txt"))
	m.files.openRow(slices.Index(shownNames(m), "a.txt"))
	if len(*opened) != 1 || m.nav.Current() != root {
		t.Errorf("removed file: opened %v, current %q", *opened, m.nav.Current())
	}
	// Opener failure is reported, not fatal.
	openFile = func(string) error { return errors.New("no xdg-open") }
	os.WriteFile(filepath.Join(root, "b"), nil, 0o644)
	wait(t, m, m.refresh)
	m.files.openRow(slices.Index(shownNames(m), "b"))
	// Out-of-range rows are ignored.
	m.files.openRow(99)
	m.clickRow(-1, 0)
}

// TC-031: grid lines are light gray, only in the table, and can be hidden.
func TestTableGridLines(t *testing.T) {
	m := newTestWindow(t, t.TempDir())
	m.app.Settings().SetTheme(newLightTheme(4.5))
	if got := (tableTheme{}).Color(theme.ColorNameSeparator, theme.VariantLight); got != gridColor {
		t.Errorf("table separator = %v", got)
	}
	if got := theme.Color(theme.ColorNameSeparator); got == gridColor {
		t.Error("app-wide separator changed too")
	}
	if got := (tableTheme{}).Color(theme.ColorNameForeground, theme.VariantLight); got != contrast.Black {
		t.Errorf("other colours not from the app theme: %v", got)
	}
	if m.files.table.HideSeparators || !m.gridItem.Checked {
		t.Fatal("grid lines hidden by default")
	}
	m.toggleGrid()
	if !m.files.table.HideSeparators || m.gridItem.Checked || !m.cfg.HideGrid {
		t.Error("toggle did not hide grid lines")
	}
	cfg := m.cfg
	m2 := newMainWindow(test.NewTempApp(t), cfg, t.TempDir())
	if !m2.files.table.HideSeparators || m2.gridItem.Checked {
		t.Error("setting not applied to a new window")
	}
}
