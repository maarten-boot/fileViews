package ui

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/maarten-boot/fileViews/internal/listing"
)

// columnWidths are the initial widths; the user can drag header borders.
var columnWidths = map[listing.Column]float32{
	listing.ColPerms:    110,
	listing.ColLinks:    55,
	listing.ColOwner:    95,
	listing.ColGroup:    95,
	listing.ColSize:     80,
	listing.ColModified: 145,
	listing.ColName:     320,
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
	m     *mainWindow
	table *widget.Table
	view  fyne.CanvasObject // the table inside its theme override
	cols  []listing.Column
}

func newFileTable(m *mainWindow) *fileTable {
	ft := &fileTable{m: m, cols: m.cfg.ColumnLayout().Visible()}
	ft.table = widget.NewTable(
		func() (int, int) { return len(m.shown), len(ft.cols) },
		func() fyne.CanvasObject {
			c := newFileCell(func(row int) { m.clickRow(row, currentModifiers()) }, ft.openRow)
			c.onSecondary = ft.showRowMenu
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
	ft.view = container.NewThemeOverride(ft.table, tableTheme{})
	return ft
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
	ft.table.Refresh()
}

func (ft *fileTable) setColumnWidths() {
	for i, c := range ft.cols {
		ft.table.SetColumnWidth(i, columnWidths[c])
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
	fi, err := os.Stat(path)
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
}

func (ft *fileTable) refresh() { ft.table.Refresh() }

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
