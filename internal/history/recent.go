package history

import "slices"

// DefaultRecentMax is the default number of recent directories kept (FR-015).
const DefaultRecentMax = 20

// Recent is a most-recent-first list of directories without duplicates (FR-015).
type Recent struct {
	items []string
	max   int
}

// NewRecent returns a list holding at most max items (DefaultRecentMax if
// max <= 0), pre-filled from items, e.g. as loaded from the config (FR-016).
// Duplicates and empty strings in items are dropped, keeping the first occurrence.
func NewRecent(max int, items []string) *Recent {
	if max <= 0 {
		max = DefaultRecentMax
	}
	r := &Recent{max: max}
	for _, dir := range items {
		if dir != "" && !slices.Contains(r.items, dir) && len(r.items) < max {
			r.items = append(r.items, dir)
		}
	}
	return r
}

// Add moves dir to the front, removing any earlier occurrence and trimming to max.
func (r *Recent) Add(dir string) {
	if i := slices.Index(r.items, dir); i >= 0 {
		r.items = slices.Delete(r.items, i, i+1)
	}
	r.items = slices.Insert(r.items, 0, dir)
	if len(r.items) > r.max {
		r.items = r.items[:r.max]
	}
}

// Items returns a copy of the list, most recent first.
func (r *Recent) Items() []string {
	return slices.Clone(r.items)
}
