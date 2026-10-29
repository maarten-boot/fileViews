// Package ui builds the Fyne user interface.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	appID    = "com.github.maarten-boot.fileviews"
	appTitle = "fileViews"
)

// Run creates the main window and blocks until the application quits.
func Run() {
	a := app.NewWithID(appID)
	newMainWindow(a).ShowAndRun()
}

// newMainWindow builds the main window for app a without showing it.
func newMainWindow(a fyne.App) fyne.Window {
	w := a.NewWindow(appTitle)
	w.SetContent(container.NewCenter(widget.NewLabel(appTitle)))
	w.Resize(fyne.NewSize(1000, 650))
	return w
}
