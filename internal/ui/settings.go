package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/maarten-boot/fileViews/internal/config"
	"github.com/maarten-boot/fileViews/internal/contrast"
	"github.com/maarten-boot/fileViews/internal/listing"
)

// columnsDialog lets the user reorder and show/hide the right-pane columns
// (FR-024). The Name column cannot be hidden.
type columnsDialog struct {
	m      *mainWindow
	order  []listing.Column
	hidden map[listing.Column]bool
	rows   *fyne.Container
	dlg    *dialog.ConfirmDialog
}

func newColumnsDialog(m *mainWindow) *columnsDialog {
	l := m.cfg.ColumnLayout()
	d := &columnsDialog{m: m, order: l.All(), hidden: map[listing.Column]bool{}, rows: container.NewVBox()}
	for c, h := range l.Hidden {
		if h && c != listing.ColName {
			d.hidden[c] = true
		}
	}
	d.build()
	content := container.NewVBox(widget.NewLabel("Tick the columns to show; use the arrows to change the order."), d.rows)
	d.dlg = dialog.NewCustomConfirm("Columns", "Apply", "Cancel", content, func(ok bool) {
		if ok {
			d.apply()
		}
	}, m.win)
	return d
}

func (d *columnsDialog) show() { d.dlg.Show() }

// build redraws one row per column: show checkbox, name, up and down.
func (d *columnsDialog) build() {
	d.rows.Objects = nil
	for i, c := range d.order {
		check := widget.NewCheck(c.String(), func(on bool) { d.setVisible(c, on) })
		check.SetChecked(!d.hidden[c])
		if c == listing.ColName {
			check.Disable()
		}
		up := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() { d.move(i, -1) })
		down := widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() { d.move(i, 1) })
		if i == 0 {
			up.Disable()
		}
		if i == len(d.order)-1 {
			down.Disable()
		}
		d.rows.Add(container.NewBorder(nil, nil, nil, container.NewHBox(up, down), check))
	}
	d.rows.Refresh()
}

// move swaps column i with its neighbour in direction step (-1 up, +1 down).
func (d *columnsDialog) move(i, step int) {
	j := i + step
	if i < 0 || j < 0 || i >= len(d.order) || j >= len(d.order) {
		return
	}
	d.order[i], d.order[j] = d.order[j], d.order[i]
	d.build()
}

func (d *columnsDialog) setVisible(c listing.Column, on bool) {
	if c == listing.ColName {
		return
	}
	if on {
		delete(d.hidden, c)
	} else {
		d.hidden[c] = true
	}
}

func (d *columnsDialog) layout() listing.ColumnLayout {
	return listing.ColumnLayout{Order: slices.Clone(d.order), Hidden: d.hidden}
}

func (d *columnsDialog) apply() {
	l := d.layout()
	d.m.cfg.SetColumnLayout(l)
	d.m.files.setColumns(l.Visible())
	d.m.changed()
}

// indentDialog sets the side pane indent per directory level (FR-044):
// a slider and a number field that follow each other.
type indentDialog struct {
	m      *mainWindow
	slider *widget.Slider
	entry  *widget.Entry
	note   *widget.Label
	apply  *widget.Button
	dlg    dialog.Dialog
	value  int
}

func newIndentDialog(m *mainWindow) *indentDialog {
	d := &indentDialog{m: m, value: m.cfg.IndentPx, note: widget.NewLabel("")}
	d.slider = widget.NewSlider(1, config.MaxIndentPx)
	d.slider.Step = 1
	d.slider.SetValue(float64(d.value))
	d.entry = widget.NewEntry()
	d.entry.SetText(strconv.Itoa(d.value))
	d.slider.OnChanged = func(v float64) {
		if int(v) != d.value || d.entry.Text != strconv.Itoa(int(v)) {
			d.value = int(v)
			d.entry.SetText(strconv.Itoa(d.value))
		}
	}
	d.entry.OnChanged = d.typed
	d.apply = widget.NewButton("Apply", func() { m.setIndent(d.value); d.dlg.Hide() })
	d.apply.Importance = widget.HighImportance
	field := container.NewBorder(nil, nil, nil, widget.NewLabel("px"), d.entry)
	row := container.NewBorder(nil, nil, nil, container.NewGridWrap(fyne.NewSize(90, d.entry.MinSize().Height), field), d.slider)
	buttons := container.NewHBox(widget.NewButton("Cancel", func() { d.dlg.Hide() }), d.apply)
	content := container.NewVBox(widget.NewLabel(fmt.Sprintf("Indent per directory level (1–%d px)", config.MaxIndentPx)), row, d.note, buttons)
	d.dlg = dialog.NewCustomWithoutButtons("Side pane indent", content, m.win)
	return d
}

func (d *indentDialog) show() { d.dlg.Show() }

// typed handles the number field: a whole number from 1 to MaxIndentPx
// moves the slider; anything else disables Apply.
func (d *indentDialog) typed(text string) {
	n, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || n < 1 || n > config.MaxIndentPx {
		d.note.SetText(fmt.Sprintf("Enter a whole number from 1 to %d.", config.MaxIndentPx))
		d.apply.Disable()
		return
	}
	d.note.SetText("")
	d.apply.Enable()
	d.value = n
	if int(d.slider.Value) != n {
		d.slider.SetValue(float64(n))
	}
}

func (m *mainWindow) showIndentDialog() { newIndentDialog(m).show() }

func (m *mainWindow) setIndent(px int) {
	m.cfg.IndentPx = px
	m.cfg.Normalize()
	m.side.refresh() // redraws and saves
}

// showContrastDialog sets the minimum contrast ratio (FR-074).
func (m *mainWindow) showContrastDialog() {
	slider, value := m.contrastSlider()
	help := widget.NewLabel("Every text and background combination must reach this contrast.\n" +
		"4.5:1 is the usual accessibility level, 7:1 the strictest.\n" +
		"Bookmarks whose colours fall below it are drawn with black or white text.")
	content := container.NewVBox(help, container.NewBorder(nil, nil, nil, value, slider))
	dialog.ShowCustomConfirm("Contrast minimum", "Apply", "Cancel", content, func(ok bool) {
		if ok {
			m.setContrastMin(slider.Value)
		}
	}, m.win)
}

func (m *mainWindow) contrastSlider() (*widget.Slider, *widget.Label) {
	slider := widget.NewSlider(contrast.FloorMin, contrast.CeilingMin)
	slider.Step = 0.5
	value := widget.NewLabel("")
	slider.OnChanged = func(v float64) { value.SetText(fmt.Sprintf("%.1f:1", v)) }
	slider.SetValue(m.cfg.ContrastMin)
	slider.OnChanged(slider.Value)
	return slider, value
}

// setContrastMin applies a new minimum: the theme gets the matching gray
// (FR-071, FR-077) and bookmarks are re-checked (FR-076).
func (m *mainWindow) setContrastMin(min float64) {
	m.cfg.ContrastMin = contrast.ClampMin(min)
	m.app.Settings().SetTheme(newLightTheme(m.cfg.ContrastMin))
	m.files.refresh()
	m.side.refresh() // redraws and saves
}
