package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/maarten-boot/fileViews/internal/history"
)

// startupDir picks the first directory to show (FR-017): the command-line
// argument, then the last directory, then home, then "/". A relative
// argument is taken from the working directory, like any command-line path.
// A non-nil error reports an unusable command-line argument; the returned
// directory is still valid.
func startupDir(arg, last, home string) (string, error) {
	var argErr error
	if arg != "" {
		var dir string
		var err error
		if filepath.IsAbs(arg) || strings.HasPrefix(arg, "~") {
			dir, err = history.NewNavigator(nil, home).Resolve(arg)
		} else {
			dir, err = filepath.Abs(arg)
		}
		if err == nil {
			err = history.CheckDir(dir)
		}
		if err == nil {
			return dir, nil
		}
		argErr = fmt.Errorf("cannot open %s: %w", arg, err)
	}
	for _, dir := range []string{last, home} {
		if dir != "" && history.CheckDir(dir) == nil {
			return dir, argErr
		}
	}
	return "/", argErr
}
