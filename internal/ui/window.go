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

// background runs slow work (file system reads) off the UI goroutine; the
// work hands its result back with fyne.Do. Tests run it inline, because
// Fyne's test driver runs fyne.Do on the calling goroutine.
var background = func(work func()) { go work() }

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
	all      []listing.Entry
	shown    []listing.Entry
	sel      selection
	drag     dragState
	free     uint64
	freeOK   bool
	loadSeq  int
	navSeq   int
	shownDir string // directory whose listing is shown

	saver        *saver // nil: nothing is saved (tests without a store)
	saveErrShown bool

	// onLoaded is called after a directory listing is shown; used by tests.
	onLoaded func()
	// onError replaces the error dialog; used by tests.
	onError func(error)
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
	m.win.SetCloseIntercept(m.quit)
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

	columns := fyne.NewMenuItem("Columns…", func() { newColumnsDialog(m).show() })
	indent := fyne.NewMenuItem("Side pane indent…", m.showIndentDialog)
	minimum := fyne.NewMenuItem("Contrast minimum…", m.showContrastDialog)

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

	m.split = container.NewHSplit(m.side.view, m.files.view)
	m.split.Offset = m.cfg.Window.Split
	return m.split
}

func (m *mainWindow) buildStatusBar() fyne.CanvasObject {
	m.status = widget.NewLabel("")
	return m.status
}

// Navigation. Each action asks the Navigator, which checks the target
// first; on error the current directory stays as it is (FR-011, FR-012).

func (m *mainWindow) goPath(p string) { m.navigate(history.MoveGo, p) }
func (m *mainWindow) goBack()         { m.navigate(history.MoveBack, "") }
func (m *mainWindow) goForward()      { m.navigate(history.MoveForward, "") }
func (m *mainWindow) goUp()           { m.navigate(history.MoveUp, "") }
func (m *mainWindow) goHome()         { m.navigate(history.MoveHome, "") }

// navigate works out the target, checks it off the UI goroutine (a hung
// network mount must not freeze the window, NFR-002) and then moves. If a
// newer navigation starts meanwhile, this one is dropped.
func (m *mainWindow) navigate(mv history.Move, path string) {
	dir, err := m.nav.Target(mv, path)
	if err != nil {
		m.navFailed(err)
		return
	}
	m.navSeq++
	seq := m.navSeq
	background(func() {
		err := history.CheckDir(dir)
		fyne.Do(func() {
			if seq != m.navSeq {
				return
			}
			if err != nil {
				m.navFailed(err)
				return
			}
			if dir != m.nav.Current() {
				m.sel.clear()
			}
			m.nav.Commit(mv, dir)
			m.side.list.Refresh() // bold follows the current directory (FR-039)
			m.load(dir)
			m.changed() // recent list and last directory
		})
	})
}

// navFailed reports a failed navigation; the current directory stays.
func (m *mainWindow) navFailed(err error) {
	m.path.SetText(m.nav.Current()) // undo whatever was typed
	m.showError(err)
}

func (m *mainWindow) showError(err error) {
	if m.onError != nil {
		m.onError(err)
		return
	}
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
	background(func() {
		entries, err := listing.Read(dir)
		free, ferr := listing.FreeSpace(dir)
		fyne.Do(func() { m.applyLoad(seq, dir, entries, err, free, ferr == nil) })
	})
}

// applyLoad shows the result of load number seq, unless a newer load has
// started since.
func (m *mainWindow) applyLoad(seq int, dir string, entries []listing.Entry, err error, free uint64, freeOK bool) {
	if seq != m.loadSeq {
		return
	}
	if dir != m.shownDir { // a new directory starts at the top (FR-020)
		m.shownDir = dir
		m.files.scrollToStart()
	}
	if err != nil {
		m.showError(err)
		entries = nil
	}
	m.all = entries
	m.free, m.freeOK = free, freeOK
	m.applyView()
	m.side.checkMissing()
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

// recentMenu lists the recent directories; choosing one goes there (FR-015).
func (m *mainWindow) recentMenu() *fyne.Menu {
	var items []*fyne.MenuItem
	for _, dir := range m.nav.Recent() {
		items = append(items, fyne.NewMenuItem(dir, func() { m.goPath(dir) }))
	}
	return fyne.NewMenu("", items...)
}

func (m *mainWindow) showRecent() {
	menu := m.recentMenu()
	if len(menu.Items) == 0 {
		return
	}
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(m.recentBtn)
	pos.Y += m.recentBtn.Size().Height
	widget.ShowPopUpMenuAtPosition(menu, m.win.Canvas(), pos)
}

func (m *mainWindow) toggleHidden() {
	m.cfg.ShowHidden = !m.cfg.ShowHidden
	m.showHiddenItem.Checked = m.cfg.ShowHidden
	m.mainMenu.Refresh()
	m.applyView()
	m.changed()
}

func (m *mainWindow) toggleGrid() {
	m.cfg.HideGrid = !m.cfg.HideGrid
	m.gridItem.Checked = !m.cfg.HideGrid
	m.mainMenu.Refresh()
	m.files.setGridLines(!m.cfg.HideGrid)
	m.changed()
}

func (m *mainWindow) toggleExactSizes() {
	m.cfg.ExactSizes = !m.cfg.ExactSizes
	m.exactSizesItem.Checked = m.cfg.ExactSizes
	m.mainMenu.Refresh()
	m.files.refresh()
	m.updateStatus()
	m.changed()
}

func (m *mainWindow) about() {
	dialog.ShowInformation("About "+appTitle, appTitle+"\nA simple hierarchical file and directory viewer.", m.win)
}

// quit saves everything, including the window size and pane split, and
// ends the application. It also handles closing the window.
func (m *mainWindow) quit() {
	m.saveNow()
	m.app.Quit()
}
