package ui

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

// TC-N04b
func TestNewMainWindow(t *testing.T) {
	a := test.NewTempApp(t)
	w := newMainWindow(a)
	if got := w.Title(); got != appTitle {
		t.Errorf("Title() = %q, want %q", got, appTitle)
	}
	if w.Content() == nil {
		t.Error("Content() is nil")
	}
}
