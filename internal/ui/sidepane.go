package ui

import (
	"errors"
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

// tooltipDelay is how long the mouse rests on a bookmark before its hover text shows.
const tooltipDelay = 600 * time.Millisecond

// sidePane is the left pane: the bookmark list (FR-040–FR-047).
type sidePane struct {
	m        *mainWindow
	list     *widget.List
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
	sp := &sidePane{m: m, tipDelay: tooltipDelay, missing: map[string]bool{}}
	sp.list = widget.NewList(
		func() int { return len(m.marks.Items()) },
		func() fyne.CanvasObject { return newMarkItem(sp) },
		func(id widget.ListItemID, o fyne.CanvasObject) { sp.update(id, o.(*markItem)) },
	)
	return sp
}

func (sp *sidePane) update(id int, mi *markItem) {
	items := sp.m.marks.Items()
	if id < 0 || id >= len(items) {
		return
	}
	mi.set(items[id], sp.missing[items[id].Path], sp.m.cfg.ContrastMin, sp.m.cfg.IndentPx)
}

// refresh redraws the list and re-checks in the background which
// bookmark directories are missing.
func (sp *sidePane) refresh() {
	sp.list.Refresh()
	sp.checkMissing()
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
	return fyne.NewMenu("",
		fyne.NewMenuItem("Open", func() { m.goPath(it.Path) }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Set colours…", func() { newColourDialog(m, it).show() }),
		reset,
		fyne.NewMenuItem("Edit hover text…", func() { sp.showHoverTextDialog(it) }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Add current directory", func() { sp.addBookmark(m.nav.Current()) }),
		remove,
	)
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

	tip      *widget.PopUp
	hoverSeq atomic.Int64 // bumped on every enter/leave; a pending tooltip shows only if unchanged
	hoverPos fyne.Position
}

func newMarkItem(sp *sidePane) *markItem {
	mi := &markItem{
		sp:     sp,
		bg:     canvas.NewRectangle(nil),
		indent: canvas.NewRectangle(nil),
		text:   canvas.NewText("", nil),
	}
	mi.ExtendBaseWidget(mi)
	return mi
}

// set shows it with the colours from bookmarks.Item.Style (FR-044–FR-047, FR-076).
func (mi *markItem) set(it bookmarks.Item, missing bool, minContrast float64, indentPx int) {
	mi.item = it
	st := it.Style(minContrast, missing)
	mi.bg.FillColor = st.Background
	mi.text.Text = st.Label
	mi.text.Color = st.Foreground
	mi.text.TextStyle = fyne.TextStyle{Italic: st.Italic}
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

func (mi *markItem) TappedSecondary(ev *fyne.PointEvent) {
	mi.hideTip()
	widget.ShowPopUpMenuAtPosition(mi.sp.menuFor(mi.item), mi.sp.m.win.Canvas(), ev.AbsolutePosition)
}

// Hover text (FR-046): Fyne has no tooltips, so show a pop-up after the
// mouse rests on the item for a moment.

func (mi *markItem) MouseIn(ev *desktop.MouseEvent) {
	mi.hoverPos = ev.AbsolutePosition
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

func (mi *markItem) MouseMoved(ev *desktop.MouseEvent) { mi.hoverPos = ev.AbsolutePosition }

func (mi *markItem) MouseOut() {
	mi.hoverSeq.Add(1)
	mi.hideTip()
}

func (mi *markItem) showTip() {
	mi.hideTip()
	label := widget.NewLabel(mi.item.Tooltip())
	mi.tip = widget.NewPopUp(label, mi.sp.m.win.Canvas())
	mi.tip.ShowAtPosition(mi.hoverPos.Add(fyne.NewPos(12, 16)))
	if mi.sp.onTip != nil {
		mi.sp.onTip()
	}
}

func (mi *markItem) hideTip() {
	if mi.tip != nil {
		mi.tip.Hide()
		mi.tip = nil
	}
}
