package listing

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Kind is the type of a directory entry.
type Kind int

const (
	KindFile Kind = iota
	KindDir
	KindSymlink
	KindFIFO
	KindSocket
	KindDevice
	KindOther
)

// Entry describes one item in a directory.
type Entry struct {
	Name    string
	Kind    Kind
	Mode    fs.FileMode
	Size    int64
	ModTime time.Time
	Links   uint64
	Owner   string
	Group   string

	// LinkTarget is the symlink target as stored in the link; empty for non-symlinks.
	LinkTarget string
	// TargetIsDir is true for a directory or a symlink that resolves to a directory.
	TargetIsDir bool
	// Err holds the error if the entry's metadata could not be read (FR-029).
	Err error
}

// Hidden reports whether the entry is a hidden (dot) file.
func (e Entry) Hidden() bool {
	return strings.HasPrefix(e.Name, ".")
}

// IsDirLike reports whether the entry is a directory or a symlink to one.
func (e Entry) IsDirLike() bool {
	return e.TargetIsDir
}

// Read returns all entries of dir, including hidden ones, unsorted.
// An error is returned only if dir itself cannot be read; problems with
// individual entries are recorded in Entry.Err (FR-029).
func Read(dir string) ([]Entry, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(des))
	for _, de := range des {
		entries = append(entries, newEntry(dir, de))
	}
	return entries, nil
}

func newEntry(dir string, de fs.DirEntry) Entry {
	e := Entry{Name: de.Name(), Kind: kindOf(de.Type())}
	info, err := de.Info()
	if err != nil {
		e.Err = err
		e.TargetIsDir = e.Kind == KindDir
		return e
	}
	e.Mode = info.Mode()
	e.Kind = kindOf(e.Mode)
	e.Size = info.Size()
	e.ModTime = info.ModTime()
	e.Links, e.Owner, e.Group = ownership(info)

	switch e.Kind {
	case KindDir:
		e.TargetIsDir = true
	case KindSymlink:
		path := filepath.Join(dir, e.Name)
		if e.LinkTarget, err = os.Readlink(path); err != nil {
			e.Err = err
		}
		// A broken symlink is not an error; it just doesn't resolve to a directory.
		if ti, err := os.Stat(path); err == nil {
			e.TargetIsDir = ti.IsDir()
		}
	}
	return e
}

func kindOf(m fs.FileMode) Kind {
	switch {
	case m&fs.ModeSymlink != 0:
		return KindSymlink
	case m.IsDir():
		return KindDir
	case m&fs.ModeNamedPipe != 0:
		return KindFIFO
	case m&fs.ModeSocket != 0:
		return KindSocket
	case m&fs.ModeDevice != 0:
		return KindDevice
	case m.IsRegular():
		return KindFile
	default:
		return KindOther
	}
}

// Filter returns the entries to display; hidden entries are dropped unless showHidden is set (FR-021).
func Filter(entries []Entry, showHidden bool) []Entry {
	if showHidden {
		return entries
	}
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if !e.Hidden() {
			out = append(out, e)
		}
	}
	return out
}
