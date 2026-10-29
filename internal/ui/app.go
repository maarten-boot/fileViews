// Package ui builds the Fyne user interface.
package ui

import (
	"errors"
	"os"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"

	"github.com/maarten-boot/fileViews/internal/config"
)

const appID = "com.github.maarten-boot.fileviews"

// Run opens the main window and blocks until the application quits.
// dirArg is the directory given on the command line, or "".
func Run(dirArg string) {
	a := app.NewWithID(appID)

	var warnings []error
	var cfg config.Config
	if path, err := config.DefaultPath(); err != nil {
		cfg, warnings = config.Default(), append(warnings, err)
	} else {
		var warn error
		cfg, warn = config.NewStore(path).Load()
		warnings = append(warnings, warn)
	}
	a.Settings().SetTheme(newLightTheme(cfg.ContrastMin))

	home, err := os.UserHomeDir()
	if err != nil {
		home = "/"
	}
	m := newMainWindow(a, cfg, home)
	start, err := startupDir(dirArg, cfg.LastDir, home)
	warnings = append(warnings, err)
	m.goPath(start)

	if err := errors.Join(warnings...); err != nil {
		dialog.ShowError(err, m.win)
	}
	m.win.ShowAndRun()
}
