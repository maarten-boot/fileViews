package ui

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/maarten-boot/fileViews/internal/listing"
)

// columnSamples are typical values used to size the columns; longer values
// are cut off with "…" until the user drags the column wider.
var columnSamples = map[listing.Column]string{
	listing.ColPerms:    "drwxr-xr-x",
	listing.ColLinks:    "999",
	listing.ColOwner:    "username",
	listing.ColGroup:    "username",
	listing.ColSize:     "999.9M",
	listing.ColModified: "2026-10-29 11:04",
}

// neededWidth is the starting width of column c: its sample value or its
// bold header, whichever is wider, plus room for the sort arrow if c is
// the sorted column. Name gets at least minNameWidth and then fills the
// pane (fitName).
func neededWidth(c listing.Column, sorted bool) float32 {
	pad := 2 * theme.InnerPadding()
	size := theme.TextSize()
	header := fyne.MeasureText(c.String(), size, fyne.TextStyle{Bold: true}).Width + pad
	if sorted {
		header += theme.IconInlineSize() + theme.Padding()
	}
	sample := fyne.MeasureText(columnSamples[c], size, fyne.TextStyle{}).Width + pad
	w := float32(math.Ceil(float64(max(header, sample))))
	if c == listing.ColName {
		w = max(w, minNameWidth)
	}
	return w
}

// openFile opens a file with the desktop's default application (FR-028).
// It is a variable so tests can replace it.
var openFile = func(path string) error {
	cmd := exec.Command("xdg-open", path)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() // reap the process
	return nil
}

// fileTable is the right pane: an ls -lF style table (FR-020–FR-028) with
// whole-row selection and clickable column headers for sorting.
type fileTable struct {
	m       *mainWindow
	table   *widget.Table
	catcher *wheelCatcher
	view    fyne.CanvasObject // table and wheel catcher inside the theme override
	cols    []listing.Column

	// autoName is the Name width last set by fitName; a different current
	// width means the user dragged it, and fitName then leaves it alone.
	autoName float32
}

// minNameWidth is the narrowest the Name column gets when filling the pane.
const minNameWidth = 150

func newFileTable(m *mainWindow) *fileTable {
	ft := &fileTable{m: m, cols: m.cfg.ColumnLayout().Visible()}
	ft.table = widget.NewTable(
		func() (int, int) { return len(m.shown), len(ft.cols) },
		func() fyne.CanvasObject {
			c := newFileCell(func(row int) { m.clickRow(row, currentModifiers()) }, ft.openRow)
			c.onSecondary = ft.showRowMenu
			c.onDrag = ft.dragRow
			c.onDragEnd = m.endDrag
			return c
		},
		ft.updateCell,
	)
	ft.table.ShowHeaderRow = true
	ft.table.CreateHeader = func() fyne.CanvasObject {
		h := newHeaderCell(ft.sortBy)
		h.target = ft.table
		return h
	}
	ft.table.UpdateHeader = ft.updateHeader
	ft.table.HideSeparators = m.cfg.HideGrid
	ft.setColumnWidths()
	ft.catcher = newWheelCatcher(ft)
	ft.view = container.NewThemeOverride(container.New(&tableLayout{ft}, ft.table, ft.catcher), tableTheme{})
	return ft
}

// tableLayout gives the table and the wheel catcher the full area and lets
// the Name column fill the width that is left (FR-020).
type tableLayout struct{ ft *fileTable }

func (l *tableLayout) MinSize(objs []fyne.CanvasObject) fyne.Size { return objs[0].MinSize() }

func (l *tableLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
	l.ft.fitName()
}

// fitName stretches the Name column so the columns fill the pane, unless
// the user has dragged it to another width. Below minNameWidth the columns
// no longer fit and the table scrolls sideways.
func (ft *fileTable) fitName() {
	i := slices.Index(ft.cols, listing.ColName)
	_, view, content, ok := tableScroller(ft.table)
	if i < 0 || !ok || view.Size().Width <= 0 {
		return
	}
	cur, known := tableColumnWidth(ft.table, i)
	if !known {
		cur = neededWidth(listing.ColName, false)
	}
	if ft.autoName != 0 && cur != ft.autoName {
		return // dragged by the user
	}
	others := content.MinSize().Width - cur
	want := max(minNameWidth, view.Size().Width-others-1)
	if want != cur {
		ft.table.SetColumnWidth(i, want)
	}
	ft.autoName = want
}

// wheelDelta filters a mouse-wheel movement (FR-020): up/down only when
// the rows do not fit, sideways only when the columns do not fit and the
// movement is sideways or Shift is held. Fyne itself would turn an up/down
// movement into a sideways scroll when only the columns overflow.
func wheelDelta(d fyne.Delta, shift bool, content, view fyne.Size) fyne.Delta {
	if shift && d.DX == 0 {
		d.DX, d.DY = d.DY, 0
	}
	if content.Height <= view.Height {
		d.DY = 0
	}
	if content.Width <= view.Width {
		d.DX = 0
	}
	return d
}

// wheel passes a filtered wheel movement on to the table's scroller.
func (ft *fileTable) wheel(ev *fyne.ScrollEvent, shift bool) {
	scroll, view, content, ok := tableScroller(ft.table)
	if !ok {
		return
	}
	d := wheelDelta(ev.Scrolled, shift, content.MinSize(), view.Size())
	if d == (fyne.Delta{}) {
		return
	}
	scroll.Scrolled(&fyne.ScrollEvent{PointEvent: ev.PointEvent, Scrolled: d})
}

// scrollToStart shows the first rows and columns, e.g. for a new directory.
func (ft *fileTable) scrollToStart() {
	ft.table.ScrollToTop()
	ft.table.ScrollToLeading()
}

// wheelCatcher lies on top of the table and only takes mouse-wheel events
// (it is not tappable or hoverable, so clicks and hovering reach the table).
// It covers the empty area below the rows too, which belongs to Fyne's
// scroller and would otherwise get the wheel directly.
type wheelCatcher struct {
	widget.BaseWidget
	ft *fileTable
}

func newWheelCatcher(ft *fileTable) *wheelCatcher {
	c := &wheelCatcher{ft: ft}
	c.ExtendBaseWidget(c)
	return c
}

func (c *wheelCatcher) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewWithoutLayout())
}

func (c *wheelCatcher) Scrolled(ev *fyne.ScrollEvent) {
	c.ft.wheel(ev, currentModifiers()&fyne.KeyModifierShift != 0)
}

// setGridLines shows or hides the lines between cells.
func (ft *fileTable) setGridLines(show bool) {
	ft.table.HideSeparators = !show
	ft.table.Refresh()
}

// gridColor is the colour of the lines between table cells: lighter than
// the separators elsewhere in the app.
var gridColor = color.RGBA{0xe8, 0xe8, 0xe8, 0xff}

// tableTheme is the app theme with lighter grid lines, used only for the table.
type tableTheme struct{}

func (tableTheme) current() fyne.Theme { return fyne.CurrentApp().Settings().Theme() }

func (t tableTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if n == theme.ColorNameSeparator {
		return gridColor
	}
	return t.current().Color(n, v)
}
func (t tableTheme) Font(s fyne.TextStyle) fyne.Resource     { return t.current().Font(s) }
func (t tableTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return t.current().Icon(n) }
func (t tableTheme) Size(n fyne.ThemeSizeName) float32       { return t.current().Size(n) }

// setColumns changes the visible columns, e.g. after the Columns settings.
func (ft *fileTable) setColumns(cols []listing.Column) {
	ft.cols = cols
	ft.setColumnWidths()
	ft.autoName = 0
	ft.fitName()
	ft.table.Refresh()
}

func (ft *fileTable) setColumnWidths() {
	sorted := ft.m.cfg.SortSpec().Column
	for i, c := range ft.cols {
		ft.table.SetColumnWidth(i, neededWidth(c, c == sorted))
	}
}

// makeRoomForArrow widens the newly sorted column if its arrow would not fit.
func (ft *fileTable) makeRoomForArrow(c listing.Column) {
	i := slices.Index(ft.cols, c)
	if i < 0 {
		return
	}
	need := neededWidth(c, true)
	if cur, ok := tableColumnWidth(ft.table, i); ok && cur < need {
		ft.table.SetColumnWidth(i, need)
		if c == listing.ColName {
			ft.autoName = need
		}
		ft.fitName()
	}
}

func (ft *fileTable) updateCell(id widget.TableCellID, o fyne.CanvasObject) {
	cell := o.(*fileCell)
	if id.Row >= len(ft.m.shown) || id.Col >= len(ft.cols) {
		cell.set(id.Row, "", fyne.TextAlignLeading, false)
		return
	}
	col := ft.cols[id.Col]
	text := listing.FormatField(ft.m.shown[id.Row], col, listing.FormatOptions{ExactSizes: ft.m.cfg.ExactSizes})
	align := fyne.TextAlignLeading
	if col == listing.ColSize || col == listing.ColLinks {
		align = fyne.TextAlignTrailing
	}
	cell.set(id.Row, text, align, ft.m.sel.has(ft.m.shown[id.Row].Name))
}

func (ft *fileTable) updateHeader(id widget.TableCellID, o fyne.CanvasObject) {
	h := o.(*headerCell)
	if id.Col < 0 || id.Col >= len(ft.cols) {
		h.set(-1, "", 0)
		return
	}
	col := ft.cols[id.Col]
	s := ft.m.cfg.SortSpec()
	dir := 0
	if s.Column == col {
		dir = 1
		if s.Descending {
			dir = -1
		}
	}
	h.set(int(col), col.String(), dir)
}

// sortBy sorts by col; clicking the sorted column again reverses it (FR-025).
func (ft *fileTable) sortBy(col int) {
	c := listing.Column(col)
	s := ft.m.cfg.SortSpec()
	if s.Column == c {
		s.Descending = !s.Descending
	} else {
		s = listing.SortSpec{Column: c}
	}
	ft.m.cfg.SetSortSpec(s)
	ft.makeRoomForArrow(c)
	ft.m.applyView()
	ft.m.changed()
}

// openRow handles a double click: directories (and links to them) are
// opened in the pane (FR-027), regular files with the default app (FR-028).
func (ft *fileTable) openRow(row int) {
	if row < 0 || row >= len(ft.m.shown) {
		return
	}
	path := filepath.Join(ft.m.nav.Current(), ft.m.shown[row].Name)
	background(func() { // stat off the UI goroutine (NFR-002)
		fi, err := os.Stat(path)
		fyne.Do(func() {
			switch {
			case err != nil:
				ft.m.showError(err)
			case fi.IsDir():
				ft.m.goPath(path)
			case fi.Mode().IsRegular():
				if err := openFile(path); err != nil {
					ft.m.showError(fmt.Errorf("cannot open %s: %w", path, err))
				}
			default:
				ft.m.showError(fmt.Errorf("cannot open %s: not a regular file", path))
			}
		})
	})
}

// refresh redraws the table after the rows changed. The table does not
// re-lay-out its scroller, which then keeps scrolling limits from the old
// row count; refreshing the scroller updates them.
func (ft *fileTable) refresh() {
	ft.table.Refresh()
	if _, scroller, _, ok := tableScroller(ft.table); ok {
		scroller.Refresh()
	}
}

// dragRow handles a drag over the table: the first call starts a drag of
// the directories at row; it returns false if there is nothing to drag.
func (ft *fileTable) dragRow(row int, abs fyne.Position, first bool) bool {
	if first {
		paths := ft.tableDragPaths(row)
		if len(paths) == 0 {
			return false
		}
		ft.m.beginTableDrag(paths)
	}
	ft.m.dragMove(abs)
	return true
}

// rowMenu is the right-click menu of a row: directories can be bookmarked
// (FR-042). It returns nil when there is nothing to offer.
func (ft *fileTable) rowMenu(row int) *fyne.Menu {
	if row < 0 || row >= len(ft.m.shown) || !ft.m.shown[row].IsDirLike() {
		return nil
	}
	path := filepath.Join(ft.m.nav.Current(), ft.m.shown[row].Name)
	return fyne.NewMenu("", fyne.NewMenuItem("Add to bookmarks", func() { ft.m.side.addBookmark(path) }))
}

func (ft *fileTable) showRowMenu(row int, pos fyne.Position) {
	if !ft.m.sel.has(ft.m.shown[row].Name) {
		ft.m.clickRow(row, 0)
	}
	if menu := ft.rowMenu(row); menu != nil {
		widget.ShowPopUpMenuAtPosition(menu, ft.m.win.Canvas(), pos)
	}
}

// fileCell is one table cell. It handles taps itself because the window
// sends a click to a single object: if cells only handled double taps,
// single taps would never reach the table.
type fileCell struct {
	widget.BaseWidget
	bg          *canvas.Rectangle
	label       *widget.Label
	row         int
	onTap       func(row int)
	onDouble    func(row int)
	onSecondary func(row int, pos fyne.Position)
	onDrag      func(row int, abs fyne.Position, first bool) bool // false: not draggable
	onDragEnd   func()
	dragging    bool // a drag that started here is in progress
	dragIgnored bool // a drag that started here has nothing to drag
}

func newFileCell(onTap, onDouble func(int)) *fileCell {
	c := &fileCell{
		bg:       canvas.NewRectangle(nil),
		label:    widget.NewLabel(""),
		onTap:    onTap,
		onDouble: onDouble,
	}
	c.label.Truncation = fyne.TextTruncateEllipsis
	c.ExtendBaseWidget(c)
	return c
}

func (c *fileCell) set(row int, text string, align fyne.TextAlign, selected bool) {
	c.row = row
	c.label.Alignment = align
	c.label.SetText(text)
	if selected {
		c.bg.FillColor = theme.Color(theme.ColorNameSelection)
	} else {
		c.bg.FillColor = nil
	}
	c.bg.Refresh()
}

func (c *fileCell) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(c.bg, c.label))
}

func (c *fileCell) Tapped(*fyne.PointEvent)       { c.onTap(c.row) }
func (c *fileCell) DoubleTapped(*fyne.PointEvent) { c.onDouble(c.row) }

// Dragged drags directories to the bookmark pane (FR-049).
func (c *fileCell) Dragged(ev *fyne.DragEvent) {
	if c.onDrag == nil || c.dragIgnored {
		return
	}
	first := !c.dragging
	if !c.onDrag(c.row, ev.AbsolutePosition, first) {
		c.dragIgnored = true
		return
	}
	c.dragging = true
}

func (c *fileCell) DragEnd() {
	if c.dragging && c.onDragEnd != nil {
		c.onDragEnd()
	}
	c.dragging, c.dragIgnored = false, false
}

func (c *fileCell) TappedSecondary(ev *fyne.PointEvent) {
	if c.onSecondary != nil {
		c.onSecondary(c.row, ev.AbsolutePosition)
	}
}

// headerCell is a clickable column header with a sort direction icon.
type headerCell struct {
	widget.BaseWidget
	label  *widget.Label
	icon   *widget.Icon
	target desktop.Mouseable // the table; receives forwarded mouse presses
	col    int
	onTap  func(col int)
}

func newHeaderCell(onTap func(int)) *headerCell {
	h := &headerCell{label: widget.NewLabel(""), icon: widget.NewIcon(nil), col: -1, onTap: onTap}
	h.label.TextStyle.Bold = true
	h.ExtendBaseWidget(h)
	return h
}

// set shows the column name; dir is 1 for ascending, -1 for descending, 0 if not sorted.
func (h *headerCell) set(col int, text string, dir int) {
	h.col = col
	h.label.SetText(text)
	switch dir {
	case 1:
		h.icon.SetResource(theme.MoveUpIcon())
	case -1:
		h.icon.SetResource(theme.MoveDownIcon())
	default:
		h.icon.SetResource(nil)
	}
}

func (h *headerCell) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewHBox(h.label, h.icon))
}

func (h *headerCell) Tapped(*fyne.PointEvent) {
	if h.col >= 0 {
		h.onTap(h.col)
	}
}

// A mouse press goes to the innermost widget that handles clicks, which is
// this header cell, so the table would never see it and could not start
// resizing a column when its border is dragged (FR-020). Pass presses on,
// in the table's coordinates.

func (h *headerCell) MouseDown(ev *desktop.MouseEvent) {
	if fwd := h.forward(ev); fwd != nil {
		h.target.MouseDown(fwd)
	}
}

func (h *headerCell) MouseUp(ev *desktop.MouseEvent) {
	if fwd := h.forward(ev); fwd != nil {
		h.target.MouseUp(fwd)
	}
}

func (h *headerCell) forward(ev *desktop.MouseEvent) *desktop.MouseEvent {
	obj, ok := h.target.(fyne.CanvasObject)
	if !ok {
		return nil
	}
	fwd := *ev
	fwd.Position = ev.AbsolutePosition.Subtract(fyne.CurrentApp().Driver().AbsolutePositionForObject(obj))
	return &fwd
}
