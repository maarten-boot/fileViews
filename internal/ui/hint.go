package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/maarten-boot/fileViews/internal/contrast"
)

// hint is a small text box drawn on top of the window content, used for
// hover texts (FR-046) and the drag label (FR-048, FR-049). It is not a
// Fyne pop-up: while a pop-up is open Fyne sends every mouse event to the
// pop-up, so a hover text would swallow the next click. A hint takes no
// mouse events at all; clicks go to whatever lies underneath.
type hint struct {
	layer *fyne.Container // full-window layer above the content
	box   *fyne.Container
	label *widget.Label
}

func newHint(layer *fyne.Container) *hint {
	bg := canvas.NewRectangle(contrast.White)
	bg.StrokeColor = colorBorder
	bg.StrokeWidth = 1
	h := &hint{layer: layer, label: widget.NewLabel("")}
	h.box = container.NewStack(bg, h.label)
	h.box.Hide()
	layer.Add(h.box)
	return h
}

// show places the hint with text at the absolute position abs, kept
// inside the window.
func (h *hint) show(text string, abs fyne.Position) {
	h.label.SetText(text)
	size := h.box.MinSize()
	pos := abs.Subtract(fyne.CurrentApp().Driver().AbsolutePositionForObject(h.layer))
	area := h.layer.Size()
	pos.X = max(0, min(pos.X, area.Width-size.Width))
	pos.Y = max(0, min(pos.Y, area.Height-size.Height))
	h.box.Move(pos)
	h.box.Resize(size)
	h.box.Show()
	h.layer.Refresh()
}

func (h *hint) hide() {
	if h.box.Visible() {
		h.box.Hide()
		h.layer.Refresh()
	}
}

func (h *hint) visible() bool { return h.box.Visible() }

func (h *hint) text() string { return h.label.Text }
