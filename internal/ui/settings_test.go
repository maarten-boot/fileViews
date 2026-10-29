package ui

import (
	"path/filepath"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/maarten-boot/fileViews/internal/bookmarks"
	"github.com/maarten-boot/fileViews/internal/config"
	"github.com/maarten-boot/fileViews/internal/contrast"
	"github.com/maarten-boot/fileViews/internal/listing"
)

// TC-024g: the Columns dialog reorders and hides columns; Name stays.
func TestColumnsDialog(t *testing.T) {
	root := fixture(t)
	m, store := savingWindow(t, config.Default(), root)
	d := newColumnsDialog(m)
	if !slices.Equal(d.order, listing.DefaultColumns) || len(d.rows.Objects) != 7 {
		t.Fatalf("order %v rows %d", d.order, len(d.rows.Objects))
	}
	nameCheck := d.rows.Objects[6].(*fyne.Container).Objects[0].(*widget.Check)
	if !nameCheck.Disabled() || !nameCheck.Checked {
		t.Error("Name checkbox can be changed")
	}
	d.move(6, -1) // Name above Modified
	d.move(0, -1) // first row cannot go up: no-op
	d.setVisible(listing.ColLinks, false)
	d.setVisible(listing.ColGroup, false)
	d.setVisible(listing.ColGroup, true)
	d.setVisible(listing.ColName, false) // ignored
	d.apply()

	want := []listing.Column{listing.ColPerms, listing.ColOwner, listing.ColGroup, listing.ColSize, listing.ColName, listing.ColModified}
	if !slices.Equal(m.files.cols, want) {
		t.Errorf("table columns %v, want %v", m.files.cols, want)
	}
	if got := loadCfg(t, store).ColumnLayout().Visible(); !slices.Equal(got, want) {
		t.Errorf("saved columns %v", got)
	}
	// Reopening shows the hidden column, unticked, in its place.
	d2 := newColumnsDialog(m)
	if d2.order[1] != listing.ColLinks || !d2.hidden[listing.ColLinks] {
		t.Errorf("reopened: order %v hidden %v", d2.order, d2.hidden)
	}
}

// TC-044d: the indent setting redraws the side pane and is saved.
func TestIndentSetting(t *testing.T) {
	root := fixture(t)
	m, store := savingWindow(t, config.Default(), root)
	d := newIndentDialog(m)
	if d.slider.Value != 12 || d.slider.Max != 25 || d.entry.Text != "12" {
		t.Errorf("slider %v (max %v), field %q", d.slider.Value, d.slider.Max, d.entry.Text)
	}
	d.entry.SetText("10") // typed
	if d.slider.Value != 10 || d.apply.Disabled() {
		t.Errorf("typed 10: slider %v, apply disabled %v", d.slider.Value, d.apply.Disabled())
	}
	d.slider.SetValue(7) // dragged
	if d.entry.Text != "7" || d.value != 7 {
		t.Errorf("slider 7: field %q value %d", d.entry.Text, d.value)
	}
	for _, bad := range []string{"0", "26", "abc", ""} {
		d.entry.SetText(bad)
		if !d.apply.Disabled() || d.note.Text == "" {
			t.Errorf("%q accepted", bad)
		}
	}
	d.entry.SetText(" 3 ")
	test.Tap(d.apply)
	if m.cfg.IndentPx != 3 {
		t.Errorf("applied %d, want 3", m.cfg.IndentPx)
	}
	m.setIndent(20)
	depth := (bookmarks.Item{Path: root}).Depth()
	if w := markAt(m, 1).indent.MinSize().Width; w != float32(20*depth) {
		t.Errorf("home indent %v, want %v", w, 20*depth)
	}
	if loadCfg(t, store).IndentPx != 20 {
		t.Error("indent not saved")
	}
	m.setIndent(500)
	if m.cfg.IndentPx != config.MaxIndentPx {
		t.Errorf("indent %d not limited", m.cfg.IndentPx)
	}
}

// TC-074c, TC-076b: a new contrast minimum changes the theme gray and the
// bookmark fallback, and is saved.
func TestContrastSetting(t *testing.T) {
	root := fixture(t)
	sub := filepath.Join(root, "sub")
	m, store := savingWindow(t, config.Default(), root)
	m.app.Settings().SetTheme(newLightTheme(m.cfg.ContrastMin))
	m.marks.Add(sub)
	m.marks.SetColours(sub, "", "#767676", contrast.DefaultMin) // passes 4.5, fails 7
	idx := slices.IndexFunc(m.marks.Items(), func(it bookmarks.Item) bool { return it.Path == sub })

	slider, label := m.contrastSlider()
	if slider.Min != 3 || slider.Max != 7 || slider.Value != 4.5 || label.Text != "4.5:1" {
		t.Errorf("slider %v..%v = %v, label %q", slider.Min, slider.Max, slider.Value, label.Text)
	}
	if contrast.Hex(markAt(m, idx).text.Color) != "#767676" {
		t.Fatal("gray text not drawn at 4.5")
	}
	m.setContrastMin(7)
	if got := contrast.Hex(theme.Color(theme.ColorNameDisabled)); got != "#595959" {
		t.Errorf("theme gray at 7 = %s", got)
	}
	if markAt(m, idx).text.Color != contrast.Black {
		t.Error("bookmark not drawn with fallback black text")
	}
	if it, _ := m.marks.Get(sub); it.Foreground != "#767676" {
		t.Error("saved colour changed")
	}
	if c := loadCfg(t, store); c.ContrastMin != 7 {
		t.Errorf("saved minimum %v", c.ContrastMin)
	}
	m.setContrastMin(9)
	if m.cfg.ContrastMin != 7 {
		t.Errorf("minimum %v not limited", m.cfg.ContrastMin)
	}
}

// TC-001b: the Settings dialogs open from the menu.
func TestSettingsMenuEnabled(t *testing.T) {
	m := newTestWindow(t, t.TempDir())
	for _, menu := range m.mainMenu.Items {
		if menu.Label != "Settings" {
			continue
		}
		for _, it := range menu.Items {
			if it.Disabled || it.Action == nil {
				t.Errorf("%s not usable", it.Label)
				continue
			}
			it.Action() // opens the dialog without panicking
		}
	}
}
