package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/maarten-boot/fileViews/internal/config"
	"github.com/maarten-boot/fileViews/internal/listing"
)

// fixture creates root/{sub/, a.txt (100 bytes), .hidden (50 bytes)}.
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "sub"), 0o755)
	os.WriteFile(filepath.Join(root, "a.txt"), make([]byte, 100), 0o644)
	os.WriteFile(filepath.Join(root, ".hidden"), make([]byte, 50), 0o644)
	return root
}

func newTestWindow(t *testing.T, home string) *mainWindow {
	t.Helper()
	a := test.NewTempApp(t)
	return newMainWindow(a, config.Default(), home)
}

// wait runs action and waits until the resulting directory load is shown.
func wait(t *testing.T, m *mainWindow, action func()) {
	t.Helper()
	done := make(chan struct{}, 1)
	m.onLoaded = func() { done <- struct{}{} }
	action()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("directory load did not finish")
	}
	m.onLoaded = nil
}

func shownNames(m *mainWindow) []string {
	var out []string
	for _, e := range m.shown {
		out = append(out, listing.DisplayName(e))
	}
	return out
}

// TC-N04b
func TestNewMainWindow(t *testing.T) {
	m := newTestWindow(t, t.TempDir())
	if got := m.win.Title(); got != appTitle {
		t.Errorf("Title() = %q, want %q", got, appTitle)
	}
	if m.win.Content() == nil {
		t.Error("Content() is nil")
	}
}

// TC-001, TC-002
func TestMenus(t *testing.T) {
	m := newTestWindow(t, t.TempDir())
	want := map[string][]string{
		"File":     {"Quit"},
		"View":     {"Show hidden files", "Show grid lines", "Refresh (F5)"},
		"Go":       {"Back", "Forward", "Up", "Home"},
		"Settings": {"Columns…", "Side pane indent…", "Exact sizes (bytes)", "Contrast minimum…"},
		"Help":     {"About"},
	}
	var order []string
	shortcuts := map[string]string{}
	for _, menu := range m.mainMenu.Items {
		order = append(order, menu.Label)
		var labels []string
		for _, it := range menu.Items {
			labels = append(labels, it.Label)
			if it.Shortcut != nil {
				shortcuts[it.Label] = it.Shortcut.ShortcutName()
			}
		}
		if !slices.Equal(labels, want[menu.Label]) {
			t.Errorf("menu %s = %v, want %v", menu.Label, labels, want[menu.Label])
		}
	}
	if !slices.Equal(order, []string{"File", "View", "Go", "Settings", "Help"}) {
		t.Errorf("menu order %v", order)
	}
	for label, sc := range map[string]string{
		"Quit": "CustomDesktop:Control+Q", "Show hidden files": "CustomDesktop:Control+H",
		"Back": "CustomDesktop:Alt+Left", "Forward": "CustomDesktop:Alt+Right",
		"Up": "CustomDesktop:Alt+Up", "Home": "CustomDesktop:Alt+Home",
	} {
		if shortcuts[label] != sc {
			t.Errorf("%s shortcut = %q, want %q", label, shortcuts[label], sc)
		}
	}
}

// TC-010, TC-011a, TC-020: navigating shows the path, the listing and the status.
func TestNavigateShowsDirectory(t *testing.T) {
	root := fixture(t)
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) })

	if m.path.Text != root {
		t.Errorf("path bar = %q", m.path.Text)
	}
	if got := shownNames(m); !slices.Equal(got, []string{"sub/", "a.txt"}) {
		t.Errorf("shown = %v", got)
	}
	if s := m.status.Text; !strings.HasPrefix(s, "1 directory · 1 file · 1 hidden · 100 total") || !strings.Contains(s, "free") {
		t.Errorf("status = %q", s)
	}
}

// TC-011b: an invalid typed path keeps the current directory.
func TestInvalidPathKeepsCurrent(t *testing.T) {
	root := fixture(t)
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) })
	m.path.SetText(filepath.Join(root, "nope"))
	m.goPath(m.path.Text)
	if m.nav.Current() != root || m.path.Text != root {
		t.Errorf("current %q, path bar %q", m.nav.Current(), m.path.Text)
	}
}

// TC-012a, TC-013, TC-014: buttons follow the history.
func TestButtonsFollowHistory(t *testing.T) {
	root := fixture(t)
	sub := filepath.Join(root, "sub")
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) })
	if !m.back.Disabled() || !m.forward.Disabled() || m.up.Disabled() {
		t.Error("after first visit: want back/forward disabled, up enabled")
	}
	wait(t, m, func() { m.goPath(sub) })
	wait(t, m, func() { test.Tap(m.back) })
	if m.nav.Current() != root || m.back.Disabled() == false || m.forward.Disabled() {
		t.Errorf("after back: current %q back %v forward %v", m.nav.Current(), m.back.Disabled(), m.forward.Disabled())
	}
	wait(t, m, func() { test.Tap(m.forward) })
	if m.nav.Current() != sub {
		t.Errorf("after forward: %q", m.nav.Current())
	}
	wait(t, m, func() { test.Tap(m.up) })
	if m.nav.Current() != root {
		t.Errorf("after up: %q", m.nav.Current())
	}
	wait(t, m, func() { m.goPath("/") })
	if !m.up.Disabled() {
		t.Error("up enabled at /")
	}
	wait(t, m, func() { test.Tap(m.home) })
	if m.nav.Current() != root {
		t.Errorf("after home: %q", m.nav.Current())
	}
}

// TC-021b, TC-051: toggling hidden files updates the list and status.
func TestToggleHidden(t *testing.T) {
	root := fixture(t)
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) })
	m.toggleHidden()
	if got := shownNames(m); !slices.Equal(got, []string{"sub/", ".hidden", "a.txt"}) {
		t.Errorf("shown = %v", got)
	}
	if !m.showHiddenItem.Checked || !strings.HasPrefix(m.status.Text, "1 directory · 2 files · 1 hidden · 150 total") {
		t.Errorf("checked %v status %q", m.showHiddenItem.Checked, m.status.Text)
	}
	m.toggleHidden()
	if len(m.shown) != 2 || m.showHiddenItem.Checked {
		t.Errorf("hidden still shown: %v", shownNames(m))
	}
}

// TC-026b, TC-050c: exact sizes and selection in the status bar.
func TestStatusSelectionAndExactSizes(t *testing.T) {
	root := fixture(t)
	os.WriteFile(filepath.Join(root, "big"), make([]byte, 4300), 0o644)
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) })
	m.clickRow(2, 0) // sub/, a.txt, big
	if !strings.Contains(m.status.Text, "1 selected (4.2K)") {
		t.Errorf("status = %q", m.status.Text)
	}
	m.toggleExactSizes()
	if !strings.Contains(m.status.Text, "1 selected (4300)") || !strings.Contains(m.status.Text, "4400 total") {
		t.Errorf("exact status = %q", m.status.Text)
	}
	m.clearSelection()
	if strings.Contains(m.status.Text, "selected") {
		t.Errorf("selection still shown: %q", m.status.Text)
	}
}

// TC-015a: the recent list follows navigation.
func TestRecentFollowsNavigation(t *testing.T) {
	root := fixture(t)
	sub := filepath.Join(root, "sub")
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) })
	wait(t, m, func() { m.goPath(sub) })
	if got := m.nav.Recent(); !slices.Equal(got, []string{sub, root}) {
		t.Errorf("recent = %v", got)
	}
}

// TC-N02: a load that finishes after a newer one started is dropped.
func TestStaleLoadDropped(t *testing.T) {
	root := fixture(t)
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) })
	before := shownNames(m)
	m.loadSeq = 7
	m.applyLoad(6, []listing.Entry{{Name: "stale"}}, nil, 0, false)
	if got := shownNames(m); !slices.Equal(got, before) {
		t.Errorf("stale result applied: %v", got)
	}
	m.applyLoad(7, []listing.Entry{{Name: "fresh"}}, nil, 0, false)
	if got := shownNames(m); !slices.Equal(got, []string{"fresh"}) {
		t.Errorf("current result not applied: %v", got)
	}
}

// TC-050e: status bar text
func TestStatusText(t *testing.T) {
	tests := []struct {
		in   statusInfo
		want string
	}{
		{statusInfo{}, "0 directories · 0 files · 0 hidden · 0 total"},
		{statusInfo{Stats: listing.Stats{Dirs: 1, Files: 1, Hidden: 2, TotalSize: 4300}, FreeKnown: true, Free: 5 << 30},
			"1 directory · 1 file · 2 hidden · 4.2K total · 5.0G free"},
		{statusInfo{Stats: listing.Stats{Dirs: 3, Files: 12, TotalSize: 4300}, Selected: 1, SelSize: 100, Exact: true},
			"3 directories · 12 files · 0 hidden · 4300 total · 1 selected (100)"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.want {
			t.Errorf("got  %q\nwant %q", got, tt.want)
		}
	}
}

// TC-017a, TC-017b, TC-017c, TC-017d
func TestStartupDir(t *testing.T) {
	root := fixture(t)
	sub := filepath.Join(root, "sub")
	gone := filepath.Join(root, "gone")
	tests := []struct {
		name, arg, last, home, want string
		wantErr                     bool
	}{
		{"argument wins", sub, root, root, sub, false},
		{"relative argument from working directory", "sub", "", "/", sub, false},
		{"home-relative argument", "~/sub", "", root, sub, false},
		{"last directory", "", sub, root, sub, false},
		{"last gone: home", "", gone, root, root, false},
		{"bad argument: falls back, error", gone, sub, root, sub, true},
		{"nothing usable: root", "", gone, gone, "/", false},
	}
	t.Chdir(root)
	for _, tt := range tests {
		got, err := startupDir(tt.arg, tt.last, tt.home)
		if got != tt.want || (err != nil) != tt.wantErr {
			t.Errorf("%s: got %q, %v; want %q, error %v", tt.name, got, err, tt.want, tt.wantErr)
		}
	}
}

// TC-002
// F5 refreshes the current directory.
func TestF5Refresh(t *testing.T) {
	root := fixture(t)
	m := newTestWindow(t, root)
	wait(t, m, func() { m.goPath(root) })
	os.WriteFile(filepath.Join(root, "new.txt"), nil, 0o644)
	wait(t, m, func() { m.win.Canvas().(test.WindowlessCanvas).OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyF5}) })
	if !slices.Contains(shownNames(m), "new.txt") {
		t.Errorf("shown = %v", shownNames(m))
	}
}
