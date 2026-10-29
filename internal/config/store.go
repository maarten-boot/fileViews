package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrNotSaved is returned by Save after Load could not read an existing
// config file, so that file is not overwritten with defaults.
var ErrNotSaved = errors.New("config not saved: the existing config file could not be read")

// DefaultPath returns $XDG_CONFIG_HOME/fileViews/config.json, falling back
// to ~/.config/fileViews/config.json (FR-060).
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "fileViews", "config.json"), nil
}

// Store loads and saves the config file at Path.
type Store struct {
	Path       string
	readFailed bool
}

// NewStore returns a store for the config file at path.
func NewStore(path string) *Store {
	return &Store{Path: path}
}

// Load reads the config file (FR-062). It always returns a usable,
// normalized Config; a non-nil error is a warning for the user:
//   - missing file: defaults, no warning
//   - invalid JSON: the file is renamed to config.json.bak, defaults, warning
//   - unreadable file: defaults, warning, and Save refuses to overwrite it
func (s *Store) Load() (Config, error) {
	s.readFailed = false
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		s.readFailed = true
		return Default(), fmt.Errorf("cannot read config, using defaults: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		bak := s.Path + ".bak"
		if rerr := os.Rename(s.Path, bak); rerr != nil {
			s.readFailed = true
			return Default(), fmt.Errorf("invalid config (%v), using defaults; could not move it aside: %w", err, rerr)
		}
		return Default(), fmt.Errorf("invalid config moved to %s, using defaults: %w", bak, err)
	}
	c.Normalize()
	return c, nil
}

// Save writes c atomically: to a temporary file in the same directory,
// which is then renamed over the config file (FR-063).
func (s *Store) Save(c Config) (err error) {
	if s.readFailed {
		return ErrNotSaved
	}
	c.Version = CurrentVersion
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".config-*.json.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Chmod(0o644); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.Path)
}
