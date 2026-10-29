package ui

import (
	"reflect"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/maarten-boot/fileViews/internal/config"
)

// rowGap returns the space between the text of two data rows and the
// header height, as the table lays them out (test only: reads internals).
func rowGap(t *testing.T, m *mainWindow) (gap, header float32) {
	t.Helper()
	m.files.table.Refresh()
	v := reflect.ValueOf(m.files.table).Elem()
	cell := float32(v.FieldByName("cellSize").FieldByName("Height").Float())
	header = float32(v.FieldByName("headerSize").FieldByName("Height").Float())
	sep := m.files.table.Theme().Size(theme.SizeNamePadding)
	text := fyne.MeasureText("Ag", theme.TextSize(), fyne.TextStyle{}).Height
	return cell + sep - text, header
}

// TC-033a, TC-033b: normal spacing unchanged; compact spacing as configured;
// the header keeps its normal height.
func TestCompactRowSpacing(t *testing.T) {
	m := shownWindow(t, fixture(t), fyne.NewSize(1000, 650))
	normalGap, normalHeader := rowGap(t, m)
	if normalGap < 18 || normalHeader != headerHeight() {
		t.Fatalf("normal view: gap %v header %v (want ≈20, %v)", normalGap, normalHeader, headerHeight())
	}
	m.toggleCompact()
	for _, gap := range []int{config.DefaultCompactGapPx, config.MinCompactGapPx, config.MaxCompactGapPx, 7} {
		m.setCompactGap(gap)
		got, header := rowGap(t, m)
		if got != float32(gap) || header != normalHeader {
			t.Errorf("compact %d px: gap %v, header %v (want %v)", gap, got, header, normalHeader)
		}
	}
	if sep := m.files.table.Theme().Size(theme.SizeNamePadding); sep != compactSeparator {
		t.Errorf("compact grid line %v", sep)
	}
	m.toggleCompact()
	if got, _ := rowGap(t, m); got != normalGap {
		t.Errorf("back to normal: gap %v, want %v", got, normalGap)
	}
	// Elsewhere the app keeps its normal padding.
	if theme.Padding() == compactSeparator {
		t.Error("app-wide padding changed")
	}
}

// TC-033c: the label keeps its own height and is centred, so the text is
// not squeezed sideways.
func TestCompactCellLayout(t *testing.T) {
	test.NewTempApp(t)
	bg, label := canvas.NewRectangle(nil), widget.NewLabel("Ag")
	lh := label.MinSize().Height
	centreLayout{}.Layout([]fyne.CanvasObject{bg, label}, fyne.NewSize(200, lh-10))
	if bg.Size() != fyne.NewSize(200, lh-10) || label.Size().Height != lh || label.Position().Y != -5 {
		t.Errorf("bg %v, label %v at %v", bg.Size(), label.Size(), label.Position())
	}
	centreLayout{}.Layout([]fyne.CanvasObject{bg, label}, fyne.NewSize(200, lh+10))
	if label.Size().Height != lh+10 || label.Position().Y != 0 {
		t.Errorf("tall cell: label %v at %v", label.Size(), label.Position())
	}
}

// TC-033d: the spacing dialog takes 2–10 px, applies and saves; the
// compact view is restored in a new window; the left pane is unchanged.
func TestCompactSettingsSaved(t *testing.T) {
	root := fixture(t)
	m, store := savingWindow(t, config.Default(), root)
	markH := markAt(m, 0).MinSize().Height

	d := newCompactGapDialog(m)
	if d.slider.Min != 2 || d.slider.Max != 10 || d.entry.Text != "4" {
		t.Errorf("dialog %v..%v %q", d.slider.Min, d.slider.Max, d.entry.Text)
	}
	for _, bad := range []string{"1", "11", "x"} {
		d.entry.SetText(bad)
		if !d.apply.Disabled() {
			t.Errorf("%q accepted", bad)
		}
	}
	d.entry.SetText("6")
	test.Tap(d.apply)
	m.toggleCompact()
	c := loadCfg(t, store)
	if !c.Compact || c.CompactGapPx != 6 {
		t.Errorf("saved compact %v gap %d", c.Compact, c.CompactGapPx)
	}
	if markAt(m, 0).MinSize().Height != markH {
		t.Error("left pane rows changed height")
	}

	m2 := shownWindow2(t, c, root)
	if !m2.compactItem.Checked {
		t.Error("compact view not restored")
	}
	if got, _ := rowGap(t, m2); got != 6 {
		t.Errorf("restored gap %v", got)
	}
}

func shownWindow2(t *testing.T, cfg config.Config, dir string) *mainWindow {
	t.Helper()
	m := newMainWindow(test.NewTempApp(t), cfg, dir)
	m.win.Resize(fyne.NewSize(1000, 650))
	wait(t, m, func() { m.goPath(dir) })
	return m
}
