package history

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrNoBack    = errors.New("no previous directory")
	ErrNoForward = errors.New("no next directory")
	ErrAtRoot    = errors.New("already at the root directory")
)

// Navigator combines the back/forward history and the recent list, and
// checks every target before moving: on error the current directory is
// left unchanged (FR-011).
type Navigator struct {
	hist   History
	recent *Recent
	home   string
}

// NewNavigator returns a navigator with no current directory; call Go to set one.
// home is used for Home and to expand a leading "~".
func NewNavigator(recent *Recent, home string) *Navigator {
	if recent == nil {
		recent = NewRecent(0, nil)
	}
	return &Navigator{recent: recent, home: home}
}

// Current returns the current directory.
func (n *Navigator) Current() string { return n.hist.Current() }

// Recent returns the recent directories, most recent first (FR-015).
func (n *Navigator) Recent() []string { return n.recent.Items() }

func (n *Navigator) CanBack() bool    { return n.hist.CanBack() }
func (n *Navigator) CanForward() bool { return n.hist.CanForward() }

// CanUp reports whether the current directory has a parent (FR-013).
func (n *Navigator) CanUp() bool {
	_, ok := Parent(n.Current())
	return ok
}

// Go navigates to path, as typed in the path bar or chosen elsewhere.
// A relative path is resolved against the current directory and a leading
// "~" is expanded to the home directory. It returns the cleaned absolute path.
func (n *Navigator) Go(path string) (string, error) {
	dir, err := n.Resolve(path)
	if err != nil {
		return "", err
	}
	if err := CheckDir(dir); err != nil {
		return "", err
	}
	n.hist.Visit(dir)
	n.recent.Add(dir)
	return dir, nil
}

// Back moves to the previous directory, if it is still readable.
func (n *Navigator) Back() (string, error) {
	dir, ok := n.hist.PeekBack()
	if !ok {
		return "", ErrNoBack
	}
	if err := CheckDir(dir); err != nil {
		return "", err
	}
	n.hist.Back()
	n.recent.Add(dir)
	return dir, nil
}

// Forward moves to the next directory, if it is still readable.
func (n *Navigator) Forward() (string, error) {
	dir, ok := n.hist.PeekForward()
	if !ok {
		return "", ErrNoForward
	}
	if err := CheckDir(dir); err != nil {
		return "", err
	}
	n.hist.Forward()
	n.recent.Add(dir)
	return dir, nil
}

// Up navigates to the parent directory (FR-013).
func (n *Navigator) Up() (string, error) {
	parent, ok := Parent(n.Current())
	if !ok {
		return "", ErrAtRoot
	}
	return n.Go(parent)
}

// Home navigates to the home directory (FR-014).
func (n *Navigator) Home() (string, error) {
	return n.Go(n.home)
}

// Resolve turns path into a cleaned absolute path without checking it.
func (n *Navigator) Resolve(path string) (string, error) {
	path = strings.TrimSpace(path)
	switch {
	case path == "":
		return "", errors.New("empty path")
	case path == "~":
		path = n.home
	case strings.HasPrefix(path, "~/"):
		path = filepath.Join(n.home, path[2:])
	}
	if !filepath.IsAbs(path) {
		base := n.Current()
		if base == "" {
			return "", fmt.Errorf("%s: relative path without a current directory", path)
		}
		path = filepath.Join(base, path)
	}
	return filepath.Clean(path), nil
}

// CheckDir returns an error unless dir exists, is a directory (or a
// symlink to one) and can be opened for reading.
func CheckDir(dir string) error {
	fi, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("%s: not a directory", dir)
	}
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	return f.Close()
}

// Parent returns the parent of dir, or false at the root or for an empty dir.
func Parent(dir string) (string, bool) {
	if dir == "" {
		return "", false
	}
	dir = filepath.Clean(dir)
	parent := filepath.Dir(dir)
	if parent == dir {
		return "", false
	}
	return parent, true
}
