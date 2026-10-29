package ui

import (
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/maarten-boot/fileViews/internal/bookmarks"
	"github.com/maarten-boot/fileViews/internal/contrast"
)

// markAt renders bookmark number id into a fresh item.
func markAt(m *mainWindow, id int) *markItem {
	mi := newMarkItem(m.side)
	m.side.update(id, mi)
	return mi
}

func menuLabels(menu *fyne.Menu) []string {
	var out []string
	for _, it := range menu.Items {
		if it.IsSeparator {
			out = append(out, "-")
			continue
		}
		label := it.Label
		if it.Disabled {
			label += " (disabled)"
		}
		out = append(out, label)
	}
	return out
}

func menuItem(t *testing.T, menu *fyne.Menu, label string) *fyne.MenuItem {
	t.Helper()
	for _, it := range menu.Items {
		if it.Label == label {
			return it
		}
	}
	t.Fatalf("menu item %q not found in %v", label, menuLabels(menu))
	return nil
}

// TC-043: clicking a bookmark opens it.
func TestBookmarkClickNavigates(t *testing.T) {
	root := fixture(t)
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(filepath.Join(root, "sub")) })
	wait(t, m, func() { test.Tap(markAt(m, 1)) }) // "/", home
	if m.nav.Current() != root {
		t.Errorf("current = %q", m.nav.Current())
	}
}

// TC-044c: the indent follows the depth and the indent setting.
func TestBookmarkIndent(t *testing.T) {
	root := fixture(t)
	m := newTestWindow(t, root)
	depth := len(strings.Split(strings.Trim(root, "/"), "/"))
	if w := markAt(m, 1).indent.MinSize().Width; w != float32(12*depth) {
		t.Errorf("indent = %v, want %v", w, 12*depth)
	}
	if w := markAt(m, 0).indent.MinSize().Width; w != 0 {
		t.Errorf("root indent = %v", w)
	}
	m.cfg.IndentPx = 20
	if w := markAt(m, 1).indent.MinSize().Width; w != float32(20*depth) {
		t.Errorf("indent at 20 px = %v", w)
	}
}

// TC-045d, TC-047c, TC-072: colours and the missing marker are drawn.
func TestBookmarkStyle(t *testing.T) {
	root := fixture(t)
	gone := filepath.Join(root, "gone")
	os.Mkdir(gone, 0o755)
	m := newTestWindow(t, root)
	m.marks.Add(gone)
	os.Remove(gone)
	m.marks.SetColours(root, "#000080", "#FFFFFF", contrast.DefaultMin)
	checkMissing(t, m)

	home := markAt(m, slices.IndexFunc(m.marks.Items(), func(it bookmarks.Item) bool { return it.Path == root }))
	if home.bg.FillColor != (contrastRGBA("#000080")) || home.text.Color != contrast.White || home.text.TextStyle.Italic {
		t.Errorf("home: bg %v text %v italic %v", home.bg.FillColor, home.text.Color, home.text.TextStyle.Italic)
	}
	slash := markAt(m, 0)
	if slash.bg.FillColor != contrast.White || slash.text.Color != contrast.Black {
		t.Error("other items changed colour too")
	}
	missing := markAt(m, slices.IndexFunc(m.marks.Items(), func(it bookmarks.Item) bool { return it.Path == gone }))
	if missing.text.Text != "gone (missing)" || !missing.text.TextStyle.Italic || contrast.Hex(missing.text.Color) != "#767676" {
		t.Errorf("missing: %q italic %v colour %s", missing.text.Text, missing.text.TextStyle.Italic, contrast.Hex(missing.text.Color))
	}
	// Clicking a missing bookmark keeps the current directory (FR-047).
	wait(t, m, func() { m.goPath(root) })
	test.Tap(missing)
	if m.nav.Current() != root {
		t.Errorf("current = %q", m.nav.Current())
	}
}

// checkMissing runs the background missing-directory check and waits for it.
func checkMissing(t *testing.T, m *mainWindow) {
	t.Helper()
	done := make(chan struct{}, 1)
	m.side.onChecked = func() { done <- struct{}{} }
	m.side.checkMissing()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("missing check did not finish")
	}
	m.side.onChecked = nil
}

// TC-047d, TC-N02: drawing uses the cached result, never a stat on the UI
// thread; the cache is refreshed in the background.
func TestBookmarkMissingCheckedInBackground(t *testing.T) {
	root := fixture(t)
	sub := filepath.Join(root, "sub")
	m := newTestWindow(t, root)
	m.marks.Add(sub)
	idx := slices.IndexFunc(m.marks.Items(), func(it bookmarks.Item) bool { return it.Path == sub })

	m.side.missing[sub] = true // stale cache: drawn as missing although it exists
	if !strings.HasSuffix(markAt(m, idx).text.Text, "(missing)") {
		t.Error("draw did not use the cache")
	}
	checkMissing(t, m)
	if strings.HasSuffix(markAt(m, idx).text.Text, "(missing)") {
		t.Error("cache not updated by the check")
	}
	os.Remove(sub)
	checkMissing(t, m)
	if !strings.HasSuffix(markAt(m, idx).text.Text, "(missing)") {
		t.Error("removed directory not detected")
	}
	// A check that finishes after a newer one started is dropped.
	m.side.missingSeq = 100
	m.side.missing = map[string]bool{}
	done := make(chan struct{}, 1)
	m.side.onChecked = func() { done <- struct{}{} }
	m.side.checkMissing() // seq 101
	<-done
	if !m.side.missing[sub] {
		t.Error("latest check not applied")
	}
}

func contrastRGBA(hex string) color.Color {
	c, _ := contrast.ParseHex(hex)
	return c
}

// TC-046d: the hover text shows after a pause and hides when the mouse leaves.
func TestBookmarkTooltip(t *testing.T) {
	root := fixture(t)
	m := newTestWindow(t, root)
	m.side.tipDelay = 0
	shown := make(chan struct{}, 1)
	m.side.onTip = func() { shown <- struct{}{} }

	mi := markAt(m, 1)
	mi.MouseIn(&desktop.MouseEvent{PointEvent: fyne.PointEvent{AbsolutePosition: fyne.NewPos(5, 5)}})
	select {
	case <-shown:
	case <-time.After(5 * time.Second):
		t.Fatal("tooltip not shown")
	}
	if mi.tip == nil || !mi.tip.Visible() {
		t.Fatal("tooltip not visible")
	}
	if got := mi.tip.Content.(*widget.Label).Text; got != root {
		t.Errorf("default tooltip = %q", got)
	}
	mi.MouseOut()
	if mi.tip != nil {
		t.Error("tooltip still shown after MouseOut")
	}

	m.side.setHoverText(root, "work")
	mi = markAt(m, 1)
	mi.showTip()
	if got := mi.tip.Content.(*widget.Label).Text; got != "work" {
		t.Errorf("custom tooltip = %q", got)
	}
}

// TC-046d: a hover that ends before the delay shows nothing.
func TestBookmarkTooltipCancelled(t *testing.T) {
	m := newTestWindow(t, t.TempDir())
	m.side.tipDelay = 50 * time.Millisecond
	mi := markAt(m, 0)
	mi.MouseIn(&desktop.MouseEvent{})
	mi.MouseOut()
	time.Sleep(150 * time.Millisecond)
	if mi.tip != nil {
		t.Error("tooltip shown after the mouse left")
	}
}

// TC-041d, TC-042c: the right-click menu and its actions.
func TestBookmarkMenu(t *testing.T) {
	root := fixture(t)
	sub := filepath.Join(root, "sub")
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(sub) })

	slash := m.marks.Items()[0]
	want := []string{"Open", "-", "Move up (disabled)", "Move down", "-", "Set colours…", "Reset colours (disabled)", "Edit hover text…", "-", "Add current directory", "Remove bookmark (disabled)"}
	if got := menuLabels(m.side.menuFor(slash)); !slices.Equal(got, want) {
		t.Errorf("menu = %v", got)
	}
	menuItem(t, m.side.menuFor(slash), "Add current directory").Action()
	it, ok := m.marks.Get(sub)
	if !ok {
		t.Fatal("current directory not added")
	}
	menuItem(t, m.side.menuFor(slash), "Add current directory").Action() // duplicate: error, no change
	if n := len(m.marks.Items()); n != 3 {
		t.Errorf("items = %d", n)
	}
	m.marks.SetColours(sub, "#FFFFE0", "#000080", contrast.DefaultMin)
	it, _ = m.marks.Get(sub)
	menu := m.side.menuFor(it)
	if menuItem(t, menu, "Reset colours").Disabled || menuItem(t, menu, "Remove bookmark").Disabled {
		t.Error("user bookmark menu items disabled")
	}
	menuItem(t, menu, "Reset colours").Action()
	if it, _ := m.marks.Get(sub); it.Background != "" || it.Foreground != "" {
		t.Errorf("not reset: %+v", it)
	}
	wait(t, m, func() { menuItem(t, menu, "Open").Action() })
	if m.nav.Current() != sub {
		t.Errorf("open: current = %q", m.nav.Current())
	}
	menuItem(t, menu, "Remove bookmark").Action()
	if _, ok := m.marks.Get(sub); ok {
		t.Error("not removed")
	}
}

// TC-042d: right-clicking a directory row offers to bookmark it.
func TestRowMenuAddBookmark(t *testing.T) {
	root := fixture(t)
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) }) // sub/, a.txt
	if m.files.rowMenu(1) != nil {
		t.Error("menu offered for a file")
	}
	menu := m.files.rowMenu(0)
	if got := menuLabels(menu); !slices.Equal(got, []string{"Add to bookmarks"}) {
		t.Fatalf("menu = %v", got)
	}
	menu.Items[0].Action()
	if _, ok := m.marks.Get(filepath.Join(root, "sub")); !ok {
		t.Error("directory not bookmarked")
	}
	m.files.showRowMenu(0, fyne.NewPos(1, 1))
	if !m.sel.has("sub") {
		t.Error("right-click did not select the row")
	}
}

// TC-075c: the colour dialog checks the ratio live.
func TestColourDialog(t *testing.T) {
	root := fixture(t)
	m := newTestWindow(t, root)
	it, _ := m.marks.Get(root)
	d := newColourDialog(m, it)

	if d.save.Disabled() || !strings.Contains(d.ratio.Text, "21.00:1") || d.suggest.Visible() {
		t.Errorf("defaults: save disabled %v, ratio %q, suggest %v", d.save.Disabled(), d.ratio.Text, d.suggest.Visible())
	}
	d.fg.SetText("#FFFF00") // yellow on white
	if !d.save.Disabled() || !strings.Contains(d.ratio.Text, "below the minimum 4.5:1") || !d.suggest.Visible() || d.suggest.Text != "Use black text" {
		t.Errorf("yellow: save disabled %v, ratio %q, suggest %q", d.save.Disabled(), d.ratio.Text, d.suggest.Text)
	}
	d.bg.SetText("#000080")
	d.fg.SetText("#000000") // black on navy
	if d.suggest.Text != "Use white text" {
		t.Errorf("suggest = %q", d.suggest.Text)
	}
	test.Tap(d.suggest)
	if d.fg.Text != "#FFFFFF" || d.save.Disabled() {
		t.Errorf("after suggestion: fg %q, save disabled %v", d.fg.Text, d.save.Disabled())
	}
	if d.previewBg.FillColor != contrastRGBA("#000080") || d.previewTxt.Color != contrast.White {
		t.Error("preview not updated")
	}
	d.bg.SetText("#12345") // invalid
	if !d.save.Disabled() || !strings.Contains(d.ratio.Text, "Invalid colour") {
		t.Errorf("invalid: %q", d.ratio.Text)
	}
	d.bg.SetText("#000080")
	test.Tap(d.save)
	if it, _ := m.marks.Get(root); it.Background != "#000080" || it.Foreground != "#FFFFFF" {
		t.Errorf("saved %+v", it)
	}
}
