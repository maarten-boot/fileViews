package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/maarten-boot/fileViews/internal/listing"
)

// selection is the set of selected entries in the right pane, by name, so
// it survives sorting, filtering and refreshing (FR-020, FR-050).
type selection struct {
	names  map[string]bool
	anchor string // last clicked name; the start of a Shift+click range
}

// click applies a click on row of shown (FR-020):
//   - plain click: select only that row
//   - Ctrl+click: add or remove that row
//   - Shift+click: select the rows from the anchor to that row
func (s *selection) click(shown []listing.Entry, row int, mod fyne.KeyModifier) {
	if row < 0 || row >= len(shown) {
		return
	}
	name := shown[row].Name
	switch {
	case mod&fyne.KeyModifierShift != 0 && s.anchor != "":
		start := indexOf(shown, s.anchor)
		if start < 0 {
			s.only(name)
			return
		}
		lo, hi := min(start, row), max(start, row)
		s.names = map[string]bool{}
		for _, e := range shown[lo : hi+1] {
			s.names[e.Name] = true
		}
	case mod&fyne.KeyModifierControl != 0:
		if s.names == nil {
			s.names = map[string]bool{}
		}
		if s.names[name] {
			delete(s.names, name)
		} else {
			s.names[name] = true
		}
		s.anchor = name
	default:
		s.only(name)
	}
}

func (s *selection) only(name string) {
	s.names = map[string]bool{name: true}
	s.anchor = name
}

func (s *selection) clear() { *s = selection{} }

func (s *selection) has(name string) bool { return s.names[name] }

// prune drops selected names that are no longer shown.
func (s *selection) prune(shown []listing.Entry) {
	visible := map[string]bool{}
	for _, e := range shown {
		visible[e.Name] = true
	}
	for name := range s.names {
		if !visible[name] {
			delete(s.names, name)
		}
	}
	if !visible[s.anchor] {
		s.anchor = ""
	}
}

// entries returns the selected entries in display order.
func (s *selection) entries(shown []listing.Entry) []listing.Entry {
	var out []listing.Entry
	for _, e := range shown {
		if s.names[e.Name] {
			out = append(out, e)
		}
	}
	return out
}

func indexOf(shown []listing.Entry, name string) int {
	for i, e := range shown {
		if e.Name == name {
			return i
		}
	}
	return -1
}

// currentModifiers returns the keyboard modifiers held down right now.
func currentModifiers() fyne.KeyModifier {
	if d, ok := fyne.CurrentApp().Driver().(desktop.Driver); ok {
		return d.CurrentKeyModifiers()
	}
	return 0
}
