package ui

import (
	"slices"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/maarten-boot/fileViews/internal/listing"
)

func entries(names ...string) []listing.Entry {
	out := make([]listing.Entry, len(names))
	for i, n := range names {
		out[i] = listing.Entry{Name: n}
	}
	return out
}

func entryNames(es []listing.Entry) []string {
	var out []string
	for _, e := range es {
		out = append(out, e.Name)
	}
	return out
}

// TC-020c
func TestSelectionClicks(t *testing.T) {
	shown := entries("a", "b", "c", "d", "e")
	var s selection
	sel := func() []string { return entryNames(s.entries(shown)) }

	s.click(shown, 1, 0)
	if !slices.Equal(sel(), []string{"b"}) {
		t.Errorf("plain click: %v", sel())
	}
	s.click(shown, 3, fyne.KeyModifierControl)
	if !slices.Equal(sel(), []string{"b", "d"}) {
		t.Errorf("ctrl add: %v", sel())
	}
	s.click(shown, 1, fyne.KeyModifierControl)
	if !slices.Equal(sel(), []string{"d"}) {
		t.Errorf("ctrl remove: %v", sel())
	}
	s.click(shown, 0, fyne.KeyModifierShift) // anchor is b (last ctrl-clicked)
	if !slices.Equal(sel(), []string{"a", "b"}) {
		t.Errorf("shift range up: %v", sel())
	}
	s.click(shown, 4, fyne.KeyModifierShift) // same anchor, range down
	if !slices.Equal(sel(), []string{"b", "c", "d", "e"}) {
		t.Errorf("shift range down: %v", sel())
	}
	s.click(shown, 2, 0)
	if !slices.Equal(sel(), []string{"c"}) {
		t.Errorf("plain click resets: %v", sel())
	}
	s.click(shown, 99, 0) // out of range: ignored
	if !slices.Equal(sel(), []string{"c"}) {
		t.Errorf("out of range: %v", sel())
	}
}

func TestSelectionShiftWithoutAnchor(t *testing.T) {
	shown := entries("a", "b", "c")
	var s selection
	s.click(shown, 2, fyne.KeyModifierShift)
	if got := entryNames(s.entries(shown)); !slices.Equal(got, []string{"c"}) {
		t.Errorf("got %v", got)
	}
}

func TestSelectionPrune(t *testing.T) {
	var s selection
	s.click(entries("a", "b", "c"), 0, 0)
	s.click(entries("a", "b", "c"), 2, fyne.KeyModifierControl)
	s.prune(entries("a", "b")) // c is no longer shown
	if s.has("c") || !s.has("a") || s.anchor != "" {
		t.Errorf("names %v anchor %q", s.names, s.anchor)
	}
	// Shift+click after the anchor disappeared behaves like a plain click.
	s.click(entries("a", "b"), 1, fyne.KeyModifierShift)
	if got := entryNames(s.entries(entries("a", "b"))); !slices.Equal(got, []string{"b"}) {
		t.Errorf("got %v", got)
	}
}
