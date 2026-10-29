package ui

import (
	"errors"
	"image/color"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/maarten-boot/fileViews/internal/bookmarks"
)

// tooltipDelay is how long the mouse must rest, without moving, on a
// bookmark before its hover text shows (FR-046).
const tooltipDelay = time.Second

// sidePane is the left pane: the bookmark list (FR-040–FR-047).
type sidePane struct {
	m        *mainWindow
	list     *widget.List
	view     fyne.CanvasObject  // list with the drop marker on top
	marker   *canvas.Rectangle  // drop position line while dragging (FR-048, FR-049)
	rendered map[*markItem]bool // item widgets created by the list, for drop positions
	tipDelay time.Duration
	onTip    func() // test hook, called after a hover text is shown

	// missing caches which bookmark directories are gone (FR-047). It is
	// filled off the UI thread: a stat on a hung network mount must not
	// freeze the window (NFR-002).
	missing    map[string]bool
	missingSeq int
	onChecked  func() // test hook, called after a missing check is applied
}

func newSidePane(m *mainWindow) *sidePane {
	sp := &sidePane{m: m, tipDelay: tooltipDelay, missing: map[string]bool{}, rendered: map[*markItem]bool{}}
	sp.list = widget.NewList(
		func() int { return len(m.marks.Items()) },
		func() fyne.CanvasObject {
			mi := newMarkItem(sp)
			sp.rendered[mi] = true
			return mi
		},
		func(id widget.ListItemID, o fyne.CanvasObject) { sp.update(id, o.(*markItem)) },
	)
	sp.marker = canvas.NewRectangle(theme.Color(theme.ColorNamePrimary))
	sp.marker.Hide()
	sp.view = container.NewStack(sp.list, container.NewWithoutLayout(sp.marker))
	return sp
}

func (sp *sidePane) update(id int, mi *markItem) {
	items := sp.m.marks.Items()
	if id < 0 || id >= len(items) {
		mi.id = -1
		return
	}
	mi.id = id
	current := items[id].Path == sp.m.nav.Current()
	mi.set(items[id], sp.missing[items[id].Path], current, sp.m.cfg.ContrastMin, sp.m.cfg.IndentPx)
}

// refresh redraws the list after the bookmarks changed, saves them and
// re-checks in the background which bookmark directories are missing.
func (sp *sidePane) refresh() {
	sp.list.Refresh()
	sp.checkMissing()
	sp.m.changed()
}

func (sp *sidePane) checkMissing() {
	sp.missingSeq++
	seq := sp.missingSeq
	items := sp.m.marks.Items()
	background(func() {
		missing := map[string]bool{}
		for _, it := range items {
			missing[it.Path] = it.Missing()
		}
		fyne.Do(func() {
			if seq != sp.missingSeq {
				return
			}
			sp.missing = missing
			sp.list.Refresh()
			if sp.onChecked != nil {
				sp.onChecked()
			}
		})
	})
}

// menuFor builds the right-click menu of a bookmark (FR-042, FR-045, FR-046).
func (sp *sidePane) menuFor(it bookmarks.Item) *fyne.Menu {
	m := sp.m
	reset := fyne.NewMenuItem("Reset colours", func() {
		m.marks.ResetColours(it.Path)
		sp.refresh()
	})
	reset.Disabled = it.Background == "" && it.Foreground == ""
	remove := fyne.NewMenuItem("Remove bookmark", func() {
		if err := m.marks.Remove(it.Path); err != nil {
			m.showError(err)
		}
		sp.refresh()
	})
	remove.Disabled = it.Default
	pos, last := m.marks.Index(it.Path), len(m.marks.Items())-1
	up := fyne.NewMenuItem("Move up", func() { sp.move(it.Path, pos-1) })
	up.Disabled = pos <= 0
	down := fyne.NewMenuItem("Move down", func() { sp.move(it.Path, pos+1) })
	down.Disabled = pos >= last
	return fyne.NewMenu("",
		fyne.NewMenuItem("Open", func() { m.goPath(it.Path) }),
		fyne.NewMenuItemSeparator(),
		up,
		down,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Set colours…", func() { newColourDialog(m, it).show() }),
		reset,
		fyne.NewMenuItem("Edit hover text…", func() { sp.showHoverTextDialog(it) }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Add current directory", func() { sp.addBookmark(m.nav.Current()) }),
		remove,
	)
}

// move puts the bookmark for path at position pos (FR-048).
func (sp *sidePane) move(path string, pos int) {
	if err := sp.m.marks.Move(path, pos); err != nil {
		sp.m.showError(err)
	}
	sp.refresh()
}

// addBookmark adds dir; adding an existing bookmark is reported (FR-042).
func (sp *sidePane) addBookmark(dir string) {
	if err := sp.m.marks.Add(dir); err != nil {
		if errors.Is(err, bookmarks.ErrExists) {
			err = errors.New(dir + " is already a bookmark")
		}
		sp.m.showError(err)
		return
	}
	sp.refresh()
}

func (sp *sidePane) showHoverTextDialog(it bookmarks.Item) {
	entry := widget.NewEntry()
	entry.SetText(it.HoverText)
	entry.SetPlaceHolder(it.Path + " (default)")
	dialog.ShowForm("Edit hover text", "Save", "Cancel",
		[]*widget.FormItem{widget.NewFormItem("Hover text", entry)},
		func(ok bool) {
			if ok {
				sp.setHoverText(it.Path, entry.Text)
			}
		}, sp.m.win)
}

func (sp *sidePane) setHoverText(path, text string) {
	if err := sp.m.marks.SetHoverText(path, text); err != nil {
		sp.m.showError(err)
	}
	sp.refresh()
}

// markItem draws one bookmark with its indent, colours and missing marker,
// and handles clicks, right-clicks and hovering itself.
type markItem struct {
	widget.BaseWidget
	sp     *sidePane
	bg     *canvas.Rectangle
	indent *canvas.Rectangle
	text   *canvas.Text
	item   bookmarks.Item
	id     int // position in the list, or -1

	dragging bool

	tipShown bool         // this item's hover text is showing
	hoverSeq atomic.Int64 // bumped on every enter, move and leave; a pending hover text shows only if unchanged
	hoverPos fyne.Position
}

func newMarkItem(sp *sidePane) *markItem {
	mi := &markItem{
		id:     -1, // until the list assigns a row; measuring templates never get one
		sp:     sp,
		bg:     canvas.NewRectangle(color.Transparent),
		indent: canvas.NewRectangle(color.Transparent),
		text:   canvas.NewText("", nil),
	}
	mi.ExtendBaseWidget(mi)
	return mi
}

// set shows it with the colours from bookmarks.Item.Style (FR-044–FR-047, FR-076).
// A current item (the directory shown in the right pane) is bold (FR-039).
func (mi *markItem) set(it bookmarks.Item, missing, current bool, minContrast float64, indentPx int) {
	mi.item = it
	st := it.Style(minContrast, missing)
	mi.bg.FillColor = st.Background
	mi.text.Text = st.Label
	mi.text.Color = st.Foreground
	mi.text.TextStyle = fyne.TextStyle{Italic: st.Italic, Bold: current}
	mi.text.TextSize = theme.TextSize()
	mi.indent.SetMinSize(fyne.NewSize(float32(it.Indent(indentPx)), 1))
	mi.Refresh()
}

func (mi *markItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(mi.bg, container.NewPadded(container.NewHBox(mi.indent, mi.text))))
}

// Tapped opens the bookmark (FR-043); a missing one gives an error (FR-047).
func (mi *markItem) Tapped(*fyne.PointEvent) {
	mi.hideTip()
	mi.sp.m.goPath(mi.item.Path)
}

// Dragged moves the bookmark to another place in the list (FR-048).
func (mi *markItem) Dragged(ev *fyne.DragEvent) {
	if mi.id < 0 {
		return
	}
	if !mi.dragging {
		mi.dragging = true
		mi.hideTip()
		mi.sp.m.beginSideDrag(mi.item.Path)
	}
	mi.sp.m.dragMove(ev.AbsolutePosition)
}

func (mi *markItem) DragEnd() {
	if mi.dragging {
		mi.dragging = false
		mi.sp.m.endDrag()
	}
}

func (mi *markItem) TappedSecondary(ev *fyne.PointEvent) {
	mi.hideTip()
	widget.ShowPopUpMenuAtPosition(mi.sp.menuFor(mi.item), mi.sp.m.win.Canvas(), ev.AbsolutePosition)
}

// Hover text (FR-046): Fyne has no tooltips. The text shows once the
// mouse has rested on the item without moving for tipDelay; any movement
// before that starts the wait again. Once shown it stays until the mouse
// leaves or clicks. It is drawn as a hint, which takes no mouse events.

func (mi *markItem) MouseIn(ev *desktop.MouseEvent) {
	mi.hoverPos = ev.AbsolutePosition
	mi.startTipTimer()
}

func (mi *markItem) MouseMoved(ev *desktop.MouseEvent) {
	if mi.tipShown {
		return
	}
	mi.hoverPos = ev.AbsolutePosition
	mi.startTipTimer()
}

func (mi *markItem) MouseOut() {
	mi.hoverSeq.Add(1)
	mi.hideTip()
}

func (mi *markItem) startTipTimer() {
	seq := mi.hoverSeq.Add(1)
	go func() {
		time.Sleep(mi.sp.tipDelay)
		fyne.Do(func() {
			if seq == mi.hoverSeq.Load() {
				mi.showTip()
			}
		})
	}()
}

func (mi *markItem) showTip() {
	mi.sp.m.tip.show(mi.item.Tooltip(), mi.hoverPos.Add(fyne.NewPos(12, 16)))
	mi.tipShown = true
	if mi.sp.onTip != nil {
		mi.sp.onTip()
	}
}

func (mi *markItem) hideTip() {
	if mi.tipShown {
		mi.hoverSeq.Add(1) // a pending hover text must not show after a click
		mi.sp.m.tip.hide()
		mi.tipShown = false
	}
}
