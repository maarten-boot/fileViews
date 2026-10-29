package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/maarten-boot/fileViews/internal/bookmarks"
	"github.com/maarten-boot/fileViews/internal/config"
	"github.com/maarten-boot/fileViews/internal/history"
	"github.com/maarten-boot/fileViews/internal/listing"
)

const appTitle = "fileViews"

// mainWindow is the single application window (§4 of the spec).
type mainWindow struct {
	app   fyne.App
	win   fyne.Window
	cfg   config.Config
	nav   *history.Navigator
	marks *bookmarks.List

	back, forward, up, home, recentBtn *widget.Button
	path                               *widget.Entry
	side                               *sidePane
	files                              *fileTable
	split                              *container.Split
	status                             *widget.Label
	showHiddenItem, exactSizesItem     *fyne.MenuItem
	gridItem                           *fyne.MenuItem
	mainMenu                           *fyne.MainMenu

	// State of the current directory; only touched on the UI goroutine.
	all     []listing.Entry
	shown   []listing.Entry
	sel     selection
	free    uint64
	freeOK  bool
	loadSeq int

	// onLoaded is called after a directory listing is shown; used by tests.
	onLoaded func()
}

func newMainWindow(a fyne.App, cfg config.Config, home string) *mainWindow {
	m := &mainWindow{
		app:   a,
		win:   a.NewWindow(appTitle),
		cfg:   cfg,
		nav:   history.NewNavigator(history.NewRecent(history.DefaultRecentMax, cfg.Recent), home),
		marks: bookmarks.New(home, cfg.Bookmarks),
	}
	m.win.SetMainMenu(m.buildMenu())
	m.win.SetContent(container.NewBorder(m.buildPathBar(), m.buildStatusBar(), nil, nil, m.buildPanes()))
	m.win.Resize(fyne.NewSize(cfg.Window.Width, cfg.Window.Height))
	m.win.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyF5 {
			m.refresh()
		}
	})
	m.updateButtons()
	return m
}

// buildMenu creates the menu bar (FR-001) with its shortcuts (FR-002).
func (m *mainWindow) buildMenu() *fyne.MainMenu {
	item := func(label string, action func(), key fyne.KeyName, mod fyne.KeyModifier) *fyne.MenuItem {
		it := fyne.NewMenuItem(label, action)
		if key != "" {
			it.Shortcut = &desktop.CustomShortcut{KeyName: key, Modifier: mod}
		}
		return it
	}
	quit := item("Quit", m.quit, fyne.KeyQ, fyne.KeyModifierControl)
	quit.IsQuit = true
	m.showHiddenItem = item("Show hidden files", m.toggleHidden, fyne.KeyH, fyne.KeyModifierControl)
	m.showHiddenItem.Checked = m.cfg.ShowHidden
	m.gridItem = item("Show grid lines", m.toggleGrid, "", 0)
	m.gridItem.Checked = !m.cfg.HideGrid
	m.exactSizesItem = item("Exact sizes (bytes)", m.toggleExactSizes, "", 0)
	m.exactSizesItem.Checked = m.cfg.ExactSizes
	refresh := item("Refresh", m.refresh, "", 0)
	refresh.Shortcut = nil // F5 is handled by the canvas key handler; shown in the label
	refresh.Label = "Refresh (F5)"

	// Settings dialogs come in milestone 9.
	columns := fyne.NewMenuItem("Columns…", nil)
	indent := fyne.NewMenuItem("Side pane indent…", nil)
	minimum := fyne.NewMenuItem("Contrast minimum…", nil)
	for _, it := range []*fyne.MenuItem{columns, indent, minimum} {
		it.Disabled = true
	}

	m.mainMenu = fyne.NewMainMenu(
		fyne.NewMenu("File", quit),
		fyne.NewMenu("View", m.showHiddenItem, m.gridItem, refresh),
		fyne.NewMenu("Go",
			item("Back", m.goBack, fyne.KeyLeft, fyne.KeyModifierAlt),
			item("Forward", m.goForward, fyne.KeyRight, fyne.KeyModifierAlt),
			item("Up", m.goUp, fyne.KeyUp, fyne.KeyModifierAlt),
			item("Home", m.goHome, fyne.KeyHome, fyne.KeyModifierAlt),
		),
		fyne.NewMenu("Settings", columns, indent, m.exactSizesItem, minimum),
		fyne.NewMenu("Help", fyne.NewMenuItem("About", m.about)),
	)
	return m.mainMenu
}

// buildPathBar creates the navigation buttons, path field and recent list (FR-010–FR-015).
func (m *mainWindow) buildPathBar() fyne.CanvasObject {
	m.back = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), m.goBack)
	m.forward = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), m.goForward)
	m.up = widget.NewButtonWithIcon("", theme.MoveUpIcon(), m.goUp)
	m.home = widget.NewButtonWithIcon("", theme.HomeIcon(), m.goHome)
	m.path = widget.NewEntry()
	m.path.SetPlaceHolder("Type a path and press Enter")
	m.path.OnSubmitted = m.goPath
	m.recentBtn = widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), m.showRecent)
	buttons := container.NewHBox(m.back, m.forward, m.up, m.home)
	return container.NewBorder(nil, nil, buttons, m.recentBtn, m.path)
}

// buildPanes creates the left (bookmarks) and right (listing) panes.
func (m *mainWindow) buildPanes() fyne.CanvasObject {
	m.side = newSidePane(m)
	m.files = newFileTable(m)

	m.split = container.NewHSplit(m.side.list, m.files.view)
	m.split.Offset = m.cfg.Window.Split
	return m.split
}

func (m *mainWindow) buildStatusBar() fyne.CanvasObject {
	m.status = widget.NewLabel("")
	return m.status
}

// Navigation. Each action asks the Navigator, which checks the target
// first; on error the current directory stays as it is (FR-011, FR-012).

func (m *mainWindow) goPath(p string) { m.navigate(func() (string, error) { return m.nav.Go(p) }) }
func (m *mainWindow) goBack()         { m.navigate(m.nav.Back) }
func (m *mainWindow) goForward()      { m.navigate(m.nav.Forward) }
func (m *mainWindow) goUp()           { m.navigate(m.nav.Up) }
func (m *mainWindow) goHome()         { m.navigate(m.nav.Home) }

func (m *mainWindow) navigate(move func() (string, error)) {
	prev := m.nav.Current()
	dir, err := move()
	if err != nil {
		m.path.SetText(prev) // undo whatever was typed
		m.showError(err)
		return
	}
	if dir != prev {
		m.sel.clear()
	}
	m.load(dir)
}

func (m *mainWindow) showError(err error) {
	dialog.ShowError(err, m.win)
}

func (m *mainWindow) refresh() {
	if dir := m.nav.Current(); dir != "" {
		m.load(dir)
	}
}

// load reads dir off the UI goroutine (NFR-002) and shows it when done.
// Results of an older load that finishes late are dropped.
func (m *mainWindow) load(dir string) {
	m.path.SetText(dir)
	m.updateButtons()
	m.loadSeq++
	seq := m.loadSeq
	go func() {
		entries, err := listing.Read(dir)
		free, ferr := listing.FreeSpace(dir)
		fyne.Do(func() { m.applyLoad(seq, entries, err, free, ferr == nil) })
	}()
}

// applyLoad shows the result of load number seq, unless a newer load has
// started since.
func (m *mainWindow) applyLoad(seq int, entries []listing.Entry, err error, free uint64, freeOK bool) {
	if seq != m.loadSeq {
		return
	}
	if err != nil {
		m.showError(err)
		entries = nil
	}
	m.all = entries
	m.free, m.freeOK = free, freeOK
	m.applyView()
	if m.onLoaded != nil {
		m.onLoaded()
	}
}

// applyView filters and sorts the current listing and redraws it. The
// selection keeps the selected entries that are still shown.
func (m *mainWindow) applyView() {
	m.shown = listing.Filter(m.all, m.cfg.ShowHidden)
	listing.Sort(m.shown, m.cfg.SortSpec())
	m.sel.prune(m.shown)
	m.files.refresh()
	m.updateStatus()
}

// clickRow applies a click with the given modifiers to a row (FR-020).
func (m *mainWindow) clickRow(row int, mod fyne.KeyModifier) {
	m.sel.click(m.shown, row, mod)
	m.files.refresh()
	m.updateStatus()
}

func (m *mainWindow) clearSelection() {
	m.sel.clear()
	m.files.refresh()
	m.updateStatus()
}

func (m *mainWindow) updateButtons() {
	setEnabled(m.back, m.nav.CanBack())
	setEnabled(m.forward, m.nav.CanForward())
	setEnabled(m.up, m.nav.CanUp())
}

func setEnabled(b *widget.Button, on bool) {
	if on {
		b.Enable()
	} else {
		b.Disable()
	}
}

func (m *mainWindow) updateStatus() {
	info := statusInfo{
		Stats:     listing.ComputeStats(m.all, m.cfg.ShowHidden),
		Free:      m.free,
		FreeKnown: m.freeOK,
		Exact:     m.cfg.ExactSizes,
	}
	if sel := m.sel.entries(m.shown); len(sel) > 0 {
		info.Selected = len(sel)
		info.SelSize = listing.TotalSize(sel)
	}
	m.status.SetText(info.String())
}

func (m *mainWindow) showRecent() {
	var items []*fyne.MenuItem
	for _, dir := range m.nav.Recent() {
		items = append(items, fyne.NewMenuItem(dir, func() { m.goPath(dir) }))
	}
	if len(items) == 0 {
		return
	}
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(m.recentBtn)
	pos.Y += m.recentBtn.Size().Height
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), m.win.Canvas(), pos)
}

func (m *mainWindow) toggleHidden() {
	m.cfg.ShowHidden = !m.cfg.ShowHidden
	m.showHiddenItem.Checked = m.cfg.ShowHidden
	m.mainMenu.Refresh()
	m.applyView()
}

func (m *mainWindow) toggleGrid() {
	m.cfg.HideGrid = !m.cfg.HideGrid
	m.gridItem.Checked = !m.cfg.HideGrid
	m.mainMenu.Refresh()
	m.files.setGridLines(!m.cfg.HideGrid)
}

func (m *mainWindow) toggleExactSizes() {
	m.cfg.ExactSizes = !m.cfg.ExactSizes
	m.exactSizesItem.Checked = m.cfg.ExactSizes
	m.mainMenu.Refresh()
	m.files.refresh()
	m.updateStatus()
}

func (m *mainWindow) about() {
	dialog.ShowInformation("About "+appTitle, appTitle+"\nA simple hierarchical file and directory viewer.", m.win)
}

func (m *mainWindow) quit() {
	m.app.Quit()
}
