package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/maarten-boot/fileViews/internal/bookmarks"
	"github.com/maarten-boot/fileViews/internal/contrast"
)

// colourDialog sets a bookmark's colours (FR-045). It shows the contrast
// ratio live and only allows saving a combination that meets the minimum;
// otherwise it offers black or white text (FR-075).
type colourDialog struct {
	m    *mainWindow
	path string

	bg, fg     *widget.Entry
	previewBg  *canvas.Rectangle
	previewTxt *canvas.Text
	ratio      *widget.Label
	save       *widget.Button
	suggest    *widget.Button
	suggestion string
	dlg        dialog.Dialog
}

func newColourDialog(m *mainWindow, it bookmarks.Item) *colourDialog {
	d := &colourDialog{
		m:          m,
		path:       it.Path,
		bg:         widget.NewEntry(),
		fg:         widget.NewEntry(),
		previewBg:  canvas.NewRectangle(contrast.White),
		previewTxt: canvas.NewText(it.Label(), contrast.Black),
		ratio:      widget.NewLabel(""),
	}
	d.bg.SetPlaceHolder("#FFFFFF (default)")
	d.fg.SetPlaceHolder("#000000 (default)")
	d.bg.SetText(it.Background)
	d.fg.SetText(it.Foreground)
	d.bg.OnChanged = func(string) { d.check() }
	d.fg.OnChanged = func(string) { d.check() }
	d.previewTxt.TextSize = theme.TextSize()
	d.save = widget.NewButton("Save", d.apply)
	d.save.Importance = widget.HighImportance
	d.suggest = widget.NewButton("", func() { d.fg.SetText(d.suggestion) })

	pick := func(e *widget.Entry, title string) *widget.Button {
		return widget.NewButton("Pick…", func() {
			p := dialog.NewColorPicker(title, "", func(c color.Color) { e.SetText(contrast.Hex(c)) }, m.win)
			p.Advanced = true
			p.Show()
		})
	}
	form := widget.NewForm(
		widget.NewFormItem("Background", container.NewBorder(nil, nil, nil, pick(d.bg, "Background colour"), d.bg)),
		widget.NewFormItem("Text", container.NewBorder(nil, nil, nil, pick(d.fg, "Text colour"), d.fg)),
	)
	preview := container.NewStack(d.previewBg, container.NewPadded(d.previewTxt))
	buttons := container.NewHBox(d.suggest, widget.NewButton("Cancel", func() { d.dlg.Hide() }), d.save)
	content := container.NewVBox(form, preview, d.ratio, buttons)
	d.dlg = dialog.NewCustomWithoutButtons("Colours for "+it.Label(), content, m.win)
	d.check()
	return d
}

func (d *colourDialog) show() { d.dlg.Show() }

// colours returns the entered colours; empty fields mean the defaults.
func (d *colourDialog) colours() (bg, fg color.RGBA, err error) {
	bg, fg = contrast.White, contrast.Black
	if d.bg.Text != "" {
		if bg, err = contrast.ParseHex(d.bg.Text); err != nil {
			return
		}
	}
	if d.fg.Text != "" {
		fg, err = contrast.ParseHex(d.fg.Text)
	}
	return
}

// check updates the preview, the ratio text and the buttons.
func (d *colourDialog) check() {
	min := d.m.cfg.ContrastMin
	bg, fg, err := d.colours()
	if err != nil {
		d.ratio.SetText("Invalid colour: use #RRGGBB")
		d.save.Disable()
		d.suggest.Hide()
		return
	}
	d.previewBg.FillColor, d.previewTxt.Color = bg, fg
	d.previewBg.Refresh()
	d.previewTxt.Refresh()
	r := contrast.Ratio(fg, bg)
	if r >= min {
		d.ratio.SetText(fmt.Sprintf("Contrast %.2f:1 — minimum %.1f:1, OK", r, min))
		d.save.Enable()
		d.suggest.Hide()
		return
	}
	d.ratio.SetText(fmt.Sprintf("Contrast %.2f:1 — below the minimum %.1f:1", r, min))
	d.save.Disable()
	d.suggestion = contrast.Hex(contrast.BestText(bg))
	if d.suggestion == "#000000" {
		d.suggest.SetText("Use black text")
	} else {
		d.suggest.SetText("Use white text")
	}
	d.suggest.Show()
}

func (d *colourDialog) apply() {
	if err := d.m.marks.SetColours(d.path, d.bg.Text, d.fg.Text, d.m.cfg.ContrastMin); err != nil {
		d.m.showError(err)
		return
	}
	d.dlg.Hide()
	d.m.side.refresh()
}
