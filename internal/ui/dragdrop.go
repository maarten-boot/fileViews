package ui

import (
	"errors"
	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"
)

// dragState is a drag in progress: a bookmark being moved within the left
// pane (FR-048), or directories dragged from the right pane to become
// bookmarks (FR-049). Fyne has no drag and drop between widgets, so the
// source widgets report the mouse and this works out the drop slot.
type dragState struct {
	active   bool
	fromSide bool
	paths    []string // the bookmark being moved, or the dragged directories
	newPaths []string // dragged directories that are not bookmarks yet
	slot     int      // insertion position in the bookmark list, or -1
}

// beginSideDrag starts moving the bookmark for path.
func (m *mainWindow) beginSideDrag(path string) {
	m.drag = dragState{active: true, fromSide: true, paths: []string{path}, slot: -1}
}

// beginTableDrag starts dragging directories from the right pane. Already
// bookmarked ones cannot be placed again (FR-049); they are only counted.
func (m *mainWindow) beginTableDrag(paths []string) {
	m.drag = dragState{active: true, paths: paths, slot: -1}
	for _, p := range paths {
		if m.marks.Index(p) < 0 {
			m.drag.newPaths = append(m.drag.newPaths, p)
		}
	}
}

// dragMove follows the mouse: it moves the floating label and the drop
// marker. Drags with nothing to add show no marker.
func (m *mainWindow) dragMove(abs fyne.Position) {
	if !m.drag.active {
		return
	}
	m.tip.hide()
	m.dragLabel.show(m.dragText(), abs.Add(fyne.NewPos(16, 12)))

	m.drag.slot = -1
	if m.drag.fromSide || len(m.drag.newPaths) > 0 {
		if slot, y, ok := m.side.slotAt(abs); ok {
			m.drag.slot = slot
			m.side.showMarker(y)
			return
		}
	}
	m.side.hideMarker()
}

// dragText describes what a drop would do.
func (m *mainWindow) dragText() string {
	d := m.drag
	if d.fromSide {
		return "Move " + filepath.Base(d.paths[0])
	}
	old := len(d.paths) - len(d.newPaths)
	switch {
	case len(d.newPaths) == 0 && len(d.paths) == 1:
		return filepath.Base(d.paths[0]) + ": already a bookmark"
	case len(d.newPaths) == 0:
		return "All already bookmarks"
	case len(d.newPaths) == 1 && old == 0:
		return "Add bookmark: " + filepath.Base(d.newPaths[0])
	case old == 0:
		return "Add " + plural(len(d.newPaths), "bookmark", "bookmarks")
	default:
		return fmt.Sprintf("Add %s (%d already bookmarked)", plural(len(d.newPaths), "bookmark", "bookmarks"), old)
	}
}

// endDrag drops at the current slot, if any, and cleans up.
func (m *mainWindow) endDrag() {
	d := m.drag
	m.drag = dragState{}
	m.dragLabel.hide()
	m.side.hideMarker()
	if !d.active || d.slot < 0 {
		return
	}
	var errs []error
	if d.fromSide {
		slot := d.slot
		if i := m.marks.Index(d.paths[0]); i >= 0 && i < slot {
			slot-- // the item leaves its old place first
		}
		errs = append(errs, m.marks.Move(d.paths[0], slot))
	} else {
		for k, p := range d.newPaths {
			errs = append(errs, m.marks.Insert(p, d.slot+k))
		}
	}
	if err := errors.Join(errs...); err != nil {
		m.showError(err)
	}
	m.side.refresh() // redraws and saves
}

// slotAt returns the insertion position for a drop at abs and the y of
// the marker line, relative to the side pane; ok is false outside the
// pane. It uses the bookmark rows Fyne has actually laid out, so it does
// not depend on guessing row heights.
func (sp *sidePane) slotAt(abs fyne.Position) (slot int, y float32, ok bool) {
	d := fyne.CurrentApp().Driver()
	listPos, size := d.AbsolutePositionForObject(sp.list), sp.list.Size()
	rel := abs.Subtract(listPos)
	if rel.X < 0 || rel.Y < 0 || rel.X > size.Width || rel.Y > size.Height {
		return 0, 0, false
	}
	n := len(sp.m.marks.Items())
	var bottom float32
	found := false
	for mi := range sp.rendered {
		if !mi.Visible() || mi.id < 0 || mi.id >= n || mi.Size().Height == 0 {
			continue
		}
		top := d.AbsolutePositionForObject(mi).Y - listPos.Y
		h := mi.Size().Height
		if top+h < 0 || top > size.Height {
			continue // scrolled out of view
		}
		found = true
		bottom = max(bottom, top+h)
		if rel.Y >= top && rel.Y < top+h {
			if rel.Y < top+h/2 {
				return mi.id, top, true
			}
			return mi.id + 1, top + h, true
		}
	}
	if !found {
		return n, 0, true
	}
	return n, bottom, true // below the last row
}

func (sp *sidePane) showMarker(y float32) {
	sp.marker.Move(fyne.NewPos(0, y-1))
	sp.marker.Resize(fyne.NewSize(sp.list.Size().Width, 2))
	sp.marker.Show()
}

func (sp *sidePane) hideMarker() { sp.marker.Hide() }

// tableDragPaths returns the directories to drag when a drag starts on
// row: the selected directories if row is selected, otherwise row itself
// (which then becomes the selection). Files are never dragged.
func (ft *fileTable) tableDragPaths(row int) []string {
	m := ft.m
	if row < 0 || row >= len(m.shown) {
		return nil
	}
	if !m.sel.has(m.shown[row].Name) {
		m.clickRow(row, 0)
	}
	var paths []string
	for _, e := range m.sel.entries(m.shown) {
		if e.IsDirLike() {
			paths = append(paths, filepath.Join(m.nav.Current(), e.Name))
		}
	}
	return paths
}
