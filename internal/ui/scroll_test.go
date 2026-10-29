package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/maarten-boot/fileViews/internal/listing"
)

// tableOffset reads the table's scroll position (test only).
func tableOffset(t *widget.Table) fyne.Position {
	v := reflect.ValueOf(t).Elem().FieldByName("offset")
	return fyne.NewPos(float32(v.FieldByName("X").Float()), float32(v.FieldByName("Y").Float()))
}

// shownWindow returns a window at the given size showing dir.
func shownWindow(t *testing.T, dir string, size fyne.Size) *mainWindow {
	t.Helper()
	m := newTestWindow(t, dir)
	m.win.Resize(size)
	wait(t, m, func() { m.goPath(dir) })
	return m
}

func wheelAt(m *mainWindow, dx, dy float32) {
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(m.files.table).Add(fyne.NewPos(100, 100))
	test.Scroll(m.win.Canvas(), pos, dx, dy)
}

// TC-020e: the reflection helpers still work with this Fyne version.
func TestFyneInternals(t *testing.T) {
	m := shownWindow(t, fixture(t), fyne.NewSize(1000, 650))
	scroll, view, content, ok := tableScroller(m.files.table)
	if !ok || scroll == nil || view.Size().Width == 0 || content.MinSize().Width == 0 {
		t.Fatal("Table scroller not found: Fyne internals changed (see fyneinternals.go)")
	}
	m.files.table.SetColumnWidth(2, 123)
	if w, ok := tableColumnWidth(m.files.table, 2); !ok || w != 123 {
		t.Fatalf("column width = %v, %v: Fyne internals changed (see fyneinternals.go)", w, ok)
	}
	if _, ok := tableColumnWidth(m.files.table, 99); ok {
		t.Error("unknown column reported")
	}
}

// TC-020f
func TestWheelDelta(t *testing.T) {
	_, tall, wide, both := fyne.NewSize(100, 100), fyne.NewSize(100, 500), fyne.NewSize(500, 100), fyne.NewSize(500, 500)
	view := fyne.NewSize(200, 200)
	tests := []struct {
		name    string
		d       fyne.Delta
		shift   bool
		content fyne.Size
		want    fyne.Delta
	}{
		{"everything fits: nothing", fyne.Delta{DY: -10}, false, fyne.NewSize(100, 100), fyne.Delta{}},
		{"only columns overflow: wheel does not go sideways", fyne.Delta{DY: -10}, false, wide, fyne.Delta{}},
		{"only columns overflow: shift goes sideways", fyne.Delta{DY: -10}, true, wide, fyne.Delta{DX: -10}},
		{"only columns overflow: sideways swipe", fyne.Delta{DX: -7}, false, wide, fyne.Delta{DX: -7}},
		{"rows overflow: up/down", fyne.Delta{DY: -10}, false, tall, fyne.Delta{DY: -10}},
		{"rows overflow: sideways swipe ignored", fyne.Delta{DX: -7, DY: -3}, false, tall, fyne.Delta{DY: -3}},
		{"both overflow: shift goes sideways", fyne.Delta{DY: -10}, true, both, fyne.Delta{DX: -10}},
	}
	for _, tt := range tests {
		if got := wheelDelta(tt.d, tt.shift, tt.content, view); got != tt.want {
			t.Errorf("%s: got %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// TC-020g: with a few rows, the mouse wheel does not move the table, also
// when the pane is too narrow for the columns.
func TestFewRowsDoNotScroll(t *testing.T) {
	root := fixture(t)
	m := shownWindow(t, root, fyne.NewSize(1000, 650))
	wheelAt(m, 0, -200)
	if off := tableOffset(m.files.table); off != (fyne.Position{}) {
		t.Errorf("wide pane: offset %v", off)
	}
	m.win.Resize(fyne.NewSize(500, 650)) // columns no longer fit
	wheelAt(m, 0, -200)
	if off := tableOffset(m.files.table); off != (fyne.Position{}) {
		t.Errorf("narrow pane, wheel: offset %v", off)
	}
	m.files.wheel(&fyne.ScrollEvent{Scrolled: fyne.Delta{DY: -50}}, true) // Shift+wheel
	if off := tableOffset(m.files.table); off.X <= 0 || off.Y != 0 {
		t.Errorf("narrow pane, shift+wheel: offset %v", off)
	}
}

// TC-020h, TC-020i: many rows scroll up and down; a new directory starts at the top.
func TestScrollManyRowsAndReset(t *testing.T) {
	big := t.TempDir()
	for i := range 200 {
		os.WriteFile(filepath.Join(big, fmt.Sprintf("f%03d", i)), nil, 0o644)
	}
	m := shownWindow(t, big, fyne.NewSize(1000, 650))
	wheelAt(m, 0, -300)
	off := tableOffset(m.files.table)
	if off.Y <= 0 || off.X != 0 {
		t.Fatalf("many rows: offset %v", off)
	}
	wait(t, m, m.refresh) // same directory: position kept
	if tableOffset(m.files.table) != off {
		t.Errorf("refresh moved the view: %v", tableOffset(m.files.table))
	}
	small := fixture(t)
	wait(t, m, func() { m.goPath(small) })
	if got := tableOffset(m.files.table); got != (fyne.Position{}) {
		t.Errorf("new directory: offset %v", got)
	}
}

// TC-020j: the Name column fills the pane, keeps a dragged width, and only
// overflows below its minimum width.
func TestNameFillsWidth(t *testing.T) {
	m := shownWindow(t, fixture(t), fyne.NewSize(1000, 650))
	fits := func() bool {
		_, view, content, _ := tableScroller(m.files.table)
		return content.MinSize().Width <= view.Size().Width
	}
	nameCol := slices.Index(m.files.cols, listing.ColName)
	w1, _ := tableColumnWidth(m.files.table, nameCol)
	if !fits() || w1 <= minNameWidth {
		t.Errorf("wide pane: fits %v, name %v", fits(), w1)
	}
	m.win.Resize(fyne.NewSize(1200, 650))
	if w2, _ := tableColumnWidth(m.files.table, nameCol); w2 <= w1 || !fits() {
		t.Errorf("wider pane: name %v (was %v), fits %v", w2, w1, fits())
	}
	m.win.Resize(fyne.NewSize(500, 650))
	if w, _ := tableColumnWidth(m.files.table, nameCol); w != minNameWidth || fits() {
		t.Errorf("narrow pane: name %v, fits %v", w, fits())
	}
	// A width set by the user (dragging) is kept when the pane changes.
	m.win.Resize(fyne.NewSize(1000, 650))
	m.files.table.SetColumnWidth(nameCol, 444)
	m.win.Resize(fyne.NewSize(1100, 650))
	if w, _ := tableColumnWidth(m.files.table, nameCol); w != 444 {
		t.Errorf("dragged width not kept: %v", w)
	}
	// Changing the columns keeps the dragged width (FR-034) ...
	m.files.setColumns(m.files.cols)
	if w, _ := tableColumnWidth(m.files.table, nameCol); w != 444 {
		t.Errorf("after setColumns: name %v, want the dragged 444", w)
	}
	// ... and resetting the widths fits Name again.
	m.files.resetWidths()
	if w, _ := tableColumnWidth(m.files.table, nameCol); w == 444 || !fits() {
		t.Errorf("after reset: name %v fits %v", w, fits())
	}
}

// TC-020k: columns start at their measured width; sorting widens a column
// that has no room for the arrow, and Name then refills the pane.
func TestColumnWidthsFromText(t *testing.T) {
	m := shownWindow(t, fixture(t), fyne.NewSize(1000, 650))
	for i, c := range m.files.cols {
		if c == listing.ColName {
			continue
		}
		w, _ := tableColumnWidth(m.files.table, i)
		if w != neededWidth(c, false) {
			t.Errorf("%s width %v, want %v", c, w, neededWidth(c, false))
		}
	}
	if neededWidth(listing.ColSize, true) <= neededWidth(listing.ColSize, false) {
		t.Error("no room reserved for the sort arrow")
	}
	sizeCol := slices.Index(m.files.cols, listing.ColSize)
	m.files.sortBy(int(listing.ColSize))
	if w, _ := tableColumnWidth(m.files.table, sizeCol); w != neededWidth(listing.ColSize, true) {
		t.Errorf("sorted Size width %v", w)
	}
	_, view, content, _ := tableScroller(m.files.table)
	if content.MinSize().Width > view.Size().Width {
		t.Error("columns no longer fit after sorting")
	}
}

// TC-020l: after a small directory, a big one still scrolls (the scroller
// must pick up the new row count).
func TestScrollAfterSmallDirectory(t *testing.T) {
	small := fixture(t)
	big := t.TempDir()
	for i := range 200 {
		os.WriteFile(filepath.Join(big, fmt.Sprintf("f%03d", i)), nil, 0o644)
	}
	m := shownWindow(t, small, fyne.NewSize(1000, 650))
	wait(t, m, func() { m.goPath(big) })
	wheelAt(m, 0, -300)
	if off := tableOffset(m.files.table); off.Y <= 0 || off.X != 0 {
		t.Errorf("offset %v", off)
	}
}
