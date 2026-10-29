package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
	cols  []listing.Column
}

func newFileTable(m *mainWindow) *fileTable {
	ft := &fileTable{m: m, cols: m.cfg.ColumnLayout().Visible()}
	ft.table = widget.NewTable(
		func() (int, int) { return len(m.shown), len(ft.cols) },
		func() fyne.CanvasObject { return newFileCell(ft.selectRow, ft.openRow) },
		ft.updateCell,
	)
	ft.table.ShowHeaderRow = true
	ft.table.CreateHeader = func() fyne.CanvasObject { return newHeaderCell(ft.sortBy) }
	ft.table.UpdateHeader = ft.updateHeader
	ft.setColumnWidths()
	return ft
}

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
	cell.set(id.Row, text, align, id.Row == ft.m.selected)
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
}

func (ft *fileTable) selectRow(row int) {
	if row < 0 || row >= len(ft.m.shown) {
		return
	}
	ft.m.setSelection(ft.m.shown[row].Name)
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

// fileCell is one table cell. It handles taps itself because the window
// sends a click to a single object: if cells only handled double taps,
// single taps would never reach the table.
type fileCell struct {
	widget.BaseWidget
	bg       *canvas.Rectangle
	label    *widget.Label
	row      int
	onTap    func(row int)
	onDouble func(row int)
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

// headerCell is a clickable column header with a sort direction icon.
type headerCell struct {
	widget.BaseWidget
	label *widget.Label
	icon  *widget.Icon
	col   int
	onTap func(col int)
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
