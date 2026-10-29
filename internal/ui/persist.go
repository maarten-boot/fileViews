package ui

import (
	"errors"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"

	"github.com/maarten-boot/fileViews/internal/config"
)

// saver writes config snapshots off the UI goroutine (FR-061, NFR-002).
// Writes are serialised, and a snapshot is never written after a newer one.
type saver struct {
	store *config.Store
	next  atomic.Int64 // sequence number of the latest snapshot handed out
	mu    sync.Mutex
	last  int64 // sequence number of the latest snapshot written
}

func newSaver(store *config.Store) *saver {
	return &saver{store: store}
}

// ticket reserves a sequence number for the snapshot taken now.
func (s *saver) ticket() int64 { return s.next.Add(1) }

// write saves snapshot seq unless a newer snapshot was already written.
func (s *saver) write(seq int64, c config.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if seq < s.last {
		return nil
	}
	s.last = seq
	return s.store.Save(c)
}

// snapshot collects everything that is kept across sessions (FR-061).
func (m *mainWindow) snapshot() config.Config {
	c := m.cfg
	c.Bookmarks = m.marks.Saved()
	c.SetColumnWidths(m.files.userWidths())
	c.Recent = m.nav.Recent()
	if dir := m.nav.Current(); dir != "" {
		c.LastDir = dir
	}
	if size := m.win.Canvas().Size(); size.Width > 0 && size.Height > 0 {
		c.Window.Width, c.Window.Height = size.Width, size.Height
	}
	c.Window.Split = m.split.Offset
	return c
}

// changed saves the current state in the background after any change to
// something that is kept across sessions.
func (m *mainWindow) changed() {
	if m.saver == nil {
		return
	}
	c, seq := m.snapshot(), m.saver.ticket()
	background(func() {
		if err := m.saver.write(seq, c); err != nil {
			fyne.Do(func() { m.saveFailed(err) })
		}
	})
}

// saveNow saves synchronously; used when quitting.
func (m *mainWindow) saveNow() {
	if m.saver == nil {
		return
	}
	if err := m.saver.write(m.saver.ticket(), m.snapshot()); err != nil {
		m.saveFailed(err)
	}
}

// saveFailed reports a save error once per session. ErrNotSaved was
// already reported as a startup warning (FR-062).
func (m *mainWindow) saveFailed(err error) {
	if m.saveErrShown || errors.Is(err, config.ErrNotSaved) {
		return
	}
	m.saveErrShown = true
	m.showError(errors.New("settings could not be saved: " + err.Error()))
}
