package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/maarten-boot/fileViews/internal/bookmarks"
)

func markPaths(m *mainWindow) []string {
	var out []string
	for _, it := range m.marks.Items() {
		out = append(out, it.Path)
	}
	return out
}

// rowPos returns an absolute point in the upper (upper=true) or lower half
// of bookmark row id, as laid out on screen.
func rowPos(t *testing.T, m *mainWindow, id int, upper bool) fyne.Position {
	t.Helper()
	for mi := range m.side.rendered {
		if mi.id == id && mi.Visible() && mi.Size().Height > 0 {
			p := fyne.CurrentApp().Driver().AbsolutePositionForObject(mi)
			y := mi.Size().Height / 4
			if !upper {
				y *= 3
			}
			return p.Add(fyne.NewPos(10, y))
		}
	}
	t.Fatalf("bookmark row %d not rendered", id)
	return fyne.Position{}
}

func dragEv(p fyne.Position) *fyne.DragEvent {
	return &fyne.DragEvent{PointEvent: fyne.PointEvent{AbsolutePosition: p}}
}

// dragWindow shows root with bookmarks /, home (=root), /a1, /a2.
func dragWindow(t *testing.T) (*mainWindow, string) {
	t.Helper()
	root := fixture(t)
	os.Mkdir(filepath.Join(root, "other"), 0o755)
	m := shownWindow(t, root, fyne.NewSize(1000, 650))
	m.marks.Add("/a1")
	m.marks.Add("/a2")
	m.side.list.Refresh()
	return m, root
}

func tableCell(m *mainWindow, row int) *fileCell {
	c := m.files.table.CreateCell().(*fileCell)
	m.files.updateCell(widget.TableCellID{Row: row, Col: 0}, c)
	return c
}

// TC-048b: Move up / Move down in the right-click menu.
func TestBookmarkMoveMenu(t *testing.T) {
	m, root := dragWindow(t) // /, root, /a1, /a2
	menuItem(t, m.side.menuFor(mustMark(t, m, "/a2")), "Move up").Action()
	menuItem(t, m.side.menuFor(mustMark(t, m, "/")), "Move down").Action()
	if got, want := markPaths(m), []string{root, "/", "/a2", "/a1"}; !slices.Equal(got, want) {
		t.Errorf("order %v, want %v", got, want)
	}
	last := m.side.menuFor(mustMark(t, m, "/a1"))
	if !menuItem(t, last, "Move down").Disabled || menuItem(t, last, "Move up").Disabled {
		t.Error("last item: Move down should be disabled, Move up enabled")
	}
}

// TC-048c: dragging a bookmark within the pane moves it.
func TestDragBookmarkWithinPane(t *testing.T) {
	m, root := dragWindow(t) // /, root, /a1, /a2
	src := itemWidget(t, m, 3)
	src.Dragged(dragEv(rowPos(t, m, 1, true))) // above root
	if !m.side.marker.Visible() || m.drag.slot != 1 {
		t.Fatalf("marker %v slot %d", m.side.marker.Visible(), m.drag.slot)
	}
	src.DragEnd()
	if got, want := markPaths(m), []string{"/", "/a2", root, "/a1"}; !slices.Equal(got, want) {
		t.Errorf("order %v, want %v", got, want)
	}
	if m.side.marker.Visible() || m.drag.active {
		t.Error("drag state not cleared")
	}
	// Moving down: drop below /a1 (the last row).
	src = itemWidget(t, m, 0)
	src.Dragged(dragEv(rowPos(t, m, 3, false)))
	src.DragEnd()
	if got, want := markPaths(m), []string{"/a2", root, "/a1", "/"}; !slices.Equal(got, want) {
		t.Errorf("order %v, want %v", got, want)
	}
}

// TC-049a: dragging a directory from the right pane adds it at the slot.
func TestDragDirectoryToPane(t *testing.T) {
	m, root := dragWindow(t)
	row := slices.Index(shownNames(m), "other/")
	c := tableCell(m, row)
	c.Dragged(dragEv(rowPos(t, m, 2, true))) // above /a1
	if !strings.HasPrefix(m.drag.label.Content.(*widget.Label).Text, "Add bookmark: other") || !m.side.marker.Visible() {
		t.Fatalf("label %q marker %v", m.drag.label.Content.(*widget.Label).Text, m.side.marker.Visible())
	}
	c.DragEnd()
	if got, want := markPaths(m), []string{"/", root, filepath.Join(root, "other"), "/a1", "/a2"}; !slices.Equal(got, want) {
		t.Errorf("order %v, want %v", got, want)
	}
}

// TC-049b: an already bookmarked directory cannot be placed again: while
// dragging the label says so and there is no drop marker; dropping does nothing.
func TestDragBookmarkedDirectoryRefused(t *testing.T) {
	m, root := dragWindow(t)
	sub := filepath.Join(root, "sub")
	m.marks.Add(sub)
	before := markPaths(m)
	var errs []error
	m.onError = func(err error) { errs = append(errs, err) }

	c := tableCell(m, slices.Index(shownNames(m), "sub/"))
	c.Dragged(dragEv(rowPos(t, m, 0, true)))
	if got := m.drag.label.Content.(*widget.Label).Text; got != "sub: already a bookmark" {
		t.Errorf("label %q", got)
	}
	if m.side.marker.Visible() || m.drag.slot != -1 {
		t.Error("drop marker shown for an existing bookmark")
	}
	c.DragEnd()
	if !slices.Equal(markPaths(m), before) || len(errs) != 0 {
		t.Errorf("order %v, errors %v", markPaths(m), errs)
	}
}

// TC-049c: dragging a selection adds the new directories only.
func TestDragSelection(t *testing.T) {
	m, root := dragWindow(t)
	sub, other := filepath.Join(root, "sub"), filepath.Join(root, "other")
	m.marks.Add(sub)
	names := shownNames(m) // other/, sub/, a.txt
	m.clickRow(slices.Index(names, "other/"), 0)
	m.clickRow(slices.Index(names, "sub/"), fyne.KeyModifierControl)
	m.clickRow(slices.Index(names, "a.txt"), fyne.KeyModifierControl) // files are not dragged
	c := tableCell(m, slices.Index(names, "sub/"))
	c.Dragged(dragEv(rowPos(t, m, 0, true)))
	if got := m.drag.label.Content.(*widget.Label).Text; got != "Add 1 bookmark (1 already bookmarked)" {
		t.Errorf("label %q", got)
	}
	c.DragEnd()
	if got := markPaths(m); got[0] != other || slices.Index(got, sub) != len(got)-1 {
		t.Errorf("order %v", got)
	}
}

// TC-049d: files are not dragged; dropping outside the pane does nothing.
func TestDragIgnoredCases(t *testing.T) {
	m, _ := dragWindow(t)
	before := markPaths(m)
	file := tableCell(m, slices.Index(shownNames(m), "a.txt"))
	file.Dragged(dragEv(rowPos(t, m, 0, true)))
	file.Dragged(dragEv(rowPos(t, m, 0, true)))
	file.DragEnd()
	if m.drag.active || !slices.Equal(markPaths(m), before) {
		t.Error("file drag did something")
	}
	dir := tableCell(m, slices.Index(shownNames(m), "other/"))
	outside := fyne.CurrentApp().Driver().AbsolutePositionForObject(m.files.table).Add(fyne.NewPos(50, 50))
	dir.Dragged(dragEv(outside))
	if m.side.marker.Visible() {
		t.Error("marker shown outside the pane")
	}
	dir.DragEnd()
	if !slices.Equal(markPaths(m), before) {
		t.Errorf("drop outside changed %v", markPaths(m))
	}
}

func mustMark(t *testing.T, m *mainWindow, path string) bookmarks.Item {
	t.Helper()
	it, ok := m.marks.Get(path)
	if !ok {
		t.Fatalf("%s not a bookmark", path)
	}
	return it
}

// itemWidget returns the rendered widget for bookmark row id.
func itemWidget(t *testing.T, m *mainWindow, id int) *markItem {
	t.Helper()
	for mi := range m.side.rendered {
		if mi.id == id && mi.Visible() && mi.Size().Height > 0 {
			return mi
		}
	}
	t.Fatalf("row %d not rendered", id)
	return nil
}
