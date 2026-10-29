package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/maarten-boot/fileViews/internal/config"
	"github.com/maarten-boot/fileViews/internal/contrast"
	"github.com/maarten-boot/fileViews/internal/listing"
)

// savingWindow returns a window that saves to a config file in a temp dir.
func savingWindow(t *testing.T, cfg config.Config, home string) (*mainWindow, *config.Store) {
	t.Helper()
	store := config.NewStore(filepath.Join(t.TempDir(), "fileViews", "config.json"))
	m := newMainWindow(test.NewTempApp(t), cfg, home)
	m.saver = newSaver(store)
	return m, store
}

func loadCfg(t *testing.T, store *config.Store) config.Config {
	t.Helper()
	c, err := config.NewStore(store.Path).Load()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TC-061b: every kind of change is saved straight away.
func TestSaveOnChange(t *testing.T) {
	root := fixture(t)
	sub := filepath.Join(root, "sub")
	m, store := savingWindow(t, config.Default(), root)

	wait(t, m, func() { m.goPath(root) })
	wait(t, m, func() { m.goPath(sub) })
	if c := loadCfg(t, store); c.LastDir != sub || !slices.Equal(c.Recent, []string{sub, root}) {
		t.Errorf("navigation: last %q recent %v", c.LastDir, c.Recent)
	}
	m.toggleHidden()
	m.toggleGrid()
	m.toggleExactSizes()
	m.files.sortBy(int(listing.ColSize))
	c := loadCfg(t, store)
	if !c.ShowHidden || !c.HideGrid || !c.ExactSizes || c.SortSpec() != (listing.SortSpec{Column: listing.ColSize}) {
		t.Errorf("toggles/sort not saved: %+v", c)
	}
	m.side.addBookmark(sub)
	m.side.setHoverText(sub, "work")
	m.marks.SetColours(sub, "#000080", "#FFFFFF", contrast.DefaultMin)
	m.side.refresh()
	c = loadCfg(t, store)
	i := slices.IndexFunc(c.Bookmarks, func(b config.Bookmark) bool { return b.Path == sub })
	if i < 0 || c.Bookmarks[i] != (config.Bookmark{Path: sub, Background: "#000080", Foreground: "#FFFFFF", HoverText: "work"}) {
		t.Errorf("bookmarks not saved: %+v", c.Bookmarks)
	}
}

// TC-061c: quitting saves the window size and pane split.
func TestSaveOnQuit(t *testing.T) {
	root := fixture(t)
	m, store := savingWindow(t, config.Default(), root)
	m.win.Resize(fyne.NewSize(800, 500))
	m.split.Offset = 0.4
	m.saveNow()
	c := loadCfg(t, store)
	if c.Window.Width != 800 || c.Window.Height != 500 || c.Window.Split != 0.4 {
		t.Errorf("window = %+v", c.Window)
	}
}

// TC-061d, TC-017e: a saved config is restored into a new window.
func TestRestoreFromConfig(t *testing.T) {
	root := fixture(t)
	sub := filepath.Join(root, "sub")
	cfg := config.Default()
	cfg.ShowHidden, cfg.HideGrid, cfg.ExactSizes = true, true, true
	cfg.SetSortSpec(listing.SortSpec{Column: listing.ColSize, Descending: true})
	cfg.Bookmarks = []config.Bookmark{{Path: sub, Background: "#000080", Foreground: "#FFFFFF", HoverText: "work"}}
	cfg.Recent = []string{sub, root}
	cfg.LastDir = sub
	cfg.Window = config.Window{Width: 900, Height: 600, Split: 0.35}

	m := newMainWindow(test.NewTempApp(t), cfg, root)
	if !m.showHiddenItem.Checked || m.gridItem.Checked || !m.exactSizesItem.Checked || !m.files.table.HideSeparators {
		t.Error("toggles not restored")
	}
	if m.split.Offset != 0.35 || m.win.Canvas().Size().Width != 900 {
		t.Errorf("split %v size %v", m.split.Offset, m.win.Canvas().Size())
	}
	if it, ok := m.marks.Get(sub); !ok || it.HoverText != "work" || it.Background != "#000080" {
		t.Errorf("bookmark %+v %v", it, ok)
	}
	if !slices.Equal(m.nav.Recent(), []string{sub, root}) {
		t.Errorf("recent %v", m.nav.Recent())
	}
	start, err := startupDir("", cfg.LastDir, root)
	if err != nil || start != sub {
		t.Errorf("startup dir %q %v", start, err)
	}
	wait(t, m, func() { m.goPath(root) })
	if got := shownNames(m); !slices.Equal(got, []string{"sub/", "a.txt", ".hidden"}) { // 100 > 50 bytes
		t.Errorf("hidden shown / size descending: %v", got)
	}
}

// TC-063c: a save error is reported once per session.
func TestSaveErrorReportedOnce(t *testing.T) {
	root := fixture(t)
	m := newMainWindow(test.NewTempApp(t), config.Default(), root)
	blocker := filepath.Join(t.TempDir(), "file")
	os.WriteFile(blocker, nil, 0o644)
	m.saver = newSaver(config.NewStore(filepath.Join(blocker, "config.json"))) // parent is a file
	var errs []error
	m.onError = func(err error) { errs = append(errs, err) }
	m.toggleHidden()
	m.toggleGrid()
	m.saveNow()
	if len(errs) != 1 {
		t.Errorf("errors reported %d times: %v", len(errs), errs)
	}
}

// TC-062d: after an unreadable config, nothing is saved and nothing is reported again.
func TestNoSaveAfterUnreadableConfig(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(`{"indentPx": 20}`), 0)
	store := config.NewStore(path)
	cfg, warn := store.Load()
	if warn == nil {
		t.Fatal("no warning")
	}
	m := newMainWindow(test.NewTempApp(t), cfg, t.TempDir())
	m.saver = newSaver(store)
	var errs []error
	m.onError = func(err error) { errs = append(errs, err) }
	m.toggleHidden()
	m.saveNow()
	os.Chmod(path, 0o644)
	if data, _ := os.ReadFile(path); string(data) != `{"indentPx": 20}` || len(errs) != 0 {
		t.Errorf("file %q, errors %v", data, errs)
	}
}

// TC-063d: an older snapshot never overwrites a newer one.
func TestSaverOrder(t *testing.T) {
	store := config.NewStore(filepath.Join(t.TempDir(), "config.json"))
	s := newSaver(store)
	older, newer := s.ticket(), s.ticket()
	c := config.Default()
	c.IndentPx = 20
	s.write(newer, c)
	c.IndentPx = 10
	s.write(older, c) // finishes late: skipped
	if got := loadCfg(t, store).IndentPx; got != 20 {
		t.Errorf("IndentPx = %d, want the newer 20", got)
	}
}

// TC-N01b: a directory with 10,000 entries is shown within a second.
func TestLargeDirectoryInWindow(t *testing.T) {
	if testing.Short() {
		t.Skip("creates 10,000 files")
	}
	dir := t.TempDir()
	for i := range 10000 {
		os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%05d", i)), nil, 0o644)
	}
	m := newTestWindow(t, dir)
	start := time.Now()
	wait(t, m, func() { m.goPath(dir) })
	m.files.table.Refresh()
	if d := time.Since(start); d > time.Second {
		t.Errorf("took %v", d)
	} else {
		t.Logf("10,000 entries shown in %v", d)
	}
	if len(m.shown) != 10000 {
		t.Errorf("shown %d", len(m.shown))
	}
}

// TC-034b: dragged column widths are saved and restored; columns that were
// not dragged are not saved, and Name keeps a dragged width on resize.
func TestColumnWidthsRemembered(t *testing.T) {
	root := fixture(t)
	m, store := savingWindow(t, config.Default(), root)
	m.win.Resize(fyne.NewSize(1000, 650))
	wait(t, m, func() { m.goPath(root) })
	col := func(m *mainWindow, c listing.Column) int { return slices.Index(m.files.cols, c) }
	m.files.table.SetColumnWidth(col(m, listing.ColName), 300) // dragged by the user
	m.files.table.SetColumnWidth(col(m, listing.ColSize), 120)
	m.quit()

	c := loadCfg(t, store)
	w := c.ColumnWidths()
	if len(w) != 2 || w[listing.ColName] != 300 || w[listing.ColSize] != 120 {
		t.Fatalf("saved widths %v", w)
	}

	m2 := newMainWindow(test.NewTempApp(t), c, root)
	m2.win.Resize(fyne.NewSize(1000, 650))
	wait(t, m2, func() { m2.goPath(root) })
	for c, want := range map[listing.Column]float32{listing.ColName: 300, listing.ColSize: 120, listing.ColPerms: neededWidth(listing.ColPerms, false)} {
		if got, _ := tableColumnWidth(m2.files.table, col(m2, c)); got != want {
			t.Errorf("restored %s = %v, want %v", c, got, want)
		}
	}
	m2.win.Resize(fyne.NewSize(1200, 650))
	if got, _ := tableColumnWidth(m2.files.table, col(m2, listing.ColName)); got != 300 {
		t.Errorf("Name refitted to %v after resize", got)
	}
}

// TC-034c: Reset column widths forgets them, also in the saved settings.
func TestResetColumnWidths(t *testing.T) {
	root := fixture(t)
	cfg := config.Default()
	cfg.SetColumnWidths(map[listing.Column]float32{listing.ColName: 300, listing.ColSize: 120})
	m, store := savingWindow(t, cfg, root)
	m.win.Resize(fyne.NewSize(1000, 650))
	wait(t, m, func() { m.goPath(root) })
	d := newColumnsDialog(m)
	test.Tap(d.reset)
	if w := loadCfg(t, store).ColumnWidths(); len(w) != 0 {
		t.Errorf("saved widths after reset: %v", w)
	}
	sizeCol := slices.Index(m.files.cols, listing.ColSize)
	if got, _ := tableColumnWidth(m.files.table, sizeCol); got != neededWidth(listing.ColSize, false) {
		t.Errorf("Size width %v after reset", got)
	}
}
