package bookmarks

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/maarten-boot/fileViews/internal/config"
	"github.com/maarten-boot/fileViews/internal/contrast"
)

var (
	ErrIsDefault   = errors.New("default bookmarks cannot be removed")
	ErrExists      = errors.New("bookmark already exists")
	ErrNotFound    = errors.New("bookmark not found")
	ErrNotAbsolute = errors.New("bookmark path must be absolute")
)

// MissingSuffix is appended to the label of a bookmark whose directory is gone (FR-047).
const MissingSuffix = " (missing)"

// LowContrastError is returned when a colour combination is below the
// minimum (FR-075). Suggested is black or white, whichever contrasts more
// with the background.
type LowContrastError struct {
	Ratio     float64
	Min       float64
	Suggested string
}

func (e *LowContrastError) Error() string {
	return fmt.Sprintf("contrast %.2f:1 is below the minimum %.1f:1; try text colour %s", e.Ratio, e.Min, e.Suggested)
}

// Item is one side pane entry (FR-040).
type Item struct {
	Path       string
	Default    bool   // "/" and $HOME: always present, cannot be removed (FR-041)
	Background string // "#RRGGBB", or "" for the theme default
	Foreground string // "#RRGGBB", or "" for the theme default
	HoverText  string // "" means the full path (FR-046)
}

// Label is "/" for the root and the base name otherwise (FR-041).
func (it Item) Label() string {
	if it.Path == "/" {
		return "/"
	}
	return filepath.Base(it.Path)
}

// Depth is the number of path components below "/" (FR-044).
func (it Item) Depth() int {
	return len(components(it.Path))
}

// Indent returns the left indent in pixels for the item (FR-044).
func (it Item) Indent(indentPx int) int {
	return it.Depth() * indentPx
}

// Tooltip returns the hover text, defaulting to the full path (FR-046).
func (it Item) Tooltip() string {
	if it.HoverText == "" {
		return it.Path
	}
	return it.HoverText
}

// Missing reports whether the item's directory no longer exists (FR-047).
func (it Item) Missing() bool {
	fi, err := os.Stat(it.Path)
	return err != nil || !fi.IsDir()
}

// Style is how an item is drawn.
type Style struct {
	Label      string
	Background color.Color
	Foreground color.Color
	Italic     bool
}

// Style returns the colours and label to draw the item with, given the
// contrast minimum and whether its directory is missing:
//   - unset or invalid colours fall back to white background, black text (FR-070)
//   - a text colour below the minimum is drawn black or white instead (FR-076)
//   - a missing item is italic with MissingSuffix, and gray only on a pure
//     white or black background (FR-047, FR-071)
func (it Item) Style(min float64, missing bool) Style {
	bg := parseOr(it.Background, contrast.White)
	fg := contrast.Effective(parseOr(it.Foreground, contrast.Black), bg, min)
	s := Style{Label: it.Label(), Background: bg, Foreground: fg}
	if missing {
		s.Label += MissingSuffix
		s.Italic = true
		if g, ok := contrast.Gray(bg, min); ok {
			s.Foreground = g
		}
	}
	return s
}

func parseOr(hex string, def color.RGBA) color.RGBA {
	if c, err := contrast.ParseHex(hex); err == nil {
		return c
	}
	return def
}

// List is the ordered set of side pane items. Items are kept sorted by
// path, component by component, so the indent reads like a tree.
type List struct {
	items []Item
}

// New builds the list from the saved bookmarks, always including "/" and
// home as default items (FR-041). Saved entries with an empty or relative
// path are dropped; duplicates keep their first settings; saved colours
// that are not valid "#RRGGBB" are cleared.
func New(home string, saved []config.Bookmark) *List {
	l := &List{}
	l.add(Item{Path: "/", Default: true})
	if home = filepath.Clean(home); filepath.IsAbs(home) && home != "/" {
		l.add(Item{Path: home, Default: true})
	}
	seen := map[string]bool{}
	for _, b := range saved {
		if !filepath.IsAbs(b.Path) {
			continue
		}
		p := filepath.Clean(b.Path)
		if seen[p] {
			continue // duplicate entry; the first one wins
		}
		seen[p] = true
		i := l.index(p)
		if i < 0 {
			l.add(Item{Path: p})
			i = l.index(p)
		}
		l.items[i].Background = validHex(b.Background)
		l.items[i].Foreground = validHex(b.Foreground)
		l.items[i].HoverText = b.HoverText
	}
	return l
}

func validHex(s string) string {
	c, err := contrast.ParseHex(s)
	if err != nil {
		return ""
	}
	return contrast.Hex(c)
}

// Items returns a copy of the items in display order.
func (l *List) Items() []Item {
	return slices.Clone(l.items)
}

// Get returns the item for path.
func (l *List) Get(path string) (Item, bool) {
	i := l.index(filepath.Clean(path))
	if i < 0 {
		return Item{}, false
	}
	return l.items[i], true
}

// Add adds a user bookmark for an absolute directory path (FR-042).
func (l *List) Add(path string) error {
	if !filepath.IsAbs(path) {
		return ErrNotAbsolute
	}
	path = filepath.Clean(path)
	if l.index(path) >= 0 {
		return ErrExists
	}
	l.add(Item{Path: path})
	return nil
}

// Remove removes a user bookmark; default items cannot be removed (FR-041, FR-042).
func (l *List) Remove(path string) error {
	i := l.index(filepath.Clean(path))
	switch {
	case i < 0:
		return ErrNotFound
	case l.items[i].Default:
		return ErrIsDefault
	}
	l.items = slices.Delete(l.items, i, i+1)
	return nil
}

// SetColours sets the background and text colours ("#RRGGBB", or "" for the
// default) of an item (FR-045). The combination must meet min, otherwise a
// *LowContrastError is returned and nothing changes (FR-075).
func (l *List) SetColours(path, background, foreground string, min float64) error {
	i := l.index(filepath.Clean(path))
	if i < 0 {
		return ErrNotFound
	}
	bg, fg := contrast.White, contrast.Black
	var err error
	if background != "" {
		if bg, err = contrast.ParseHex(background); err != nil {
			return err
		}
		background = contrast.Hex(bg)
	}
	if foreground != "" {
		if fg, err = contrast.ParseHex(foreground); err != nil {
			return err
		}
		foreground = contrast.Hex(fg)
	}
	if r := contrast.Ratio(fg, bg); r < min {
		return &LowContrastError{Ratio: r, Min: min, Suggested: contrast.Hex(contrast.BestText(bg))}
	}
	l.items[i].Background, l.items[i].Foreground = background, foreground
	return nil
}

// ResetColours returns an item to the default colours (FR-045).
func (l *List) ResetColours(path string) error {
	return l.SetColours(path, "", "", 0)
}

// SetHoverText sets the hover text; empty or blank text means the default (FR-046).
func (l *List) SetHoverText(path, text string) error {
	i := l.index(filepath.Clean(path))
	if i < 0 {
		return ErrNotFound
	}
	if strings.TrimSpace(text) == "" {
		text = ""
	}
	l.items[i].HoverText = text
	return nil
}

// Saved returns the bookmarks to store in the config (FR-045, FR-046).
func (l *List) Saved() []config.Bookmark {
	out := make([]config.Bookmark, len(l.items))
	for i, it := range l.items {
		out[i] = config.Bookmark{Path: it.Path, Background: it.Background, Foreground: it.Foreground, HoverText: it.HoverText}
	}
	return out
}

func (l *List) index(path string) int {
	return slices.IndexFunc(l.items, func(it Item) bool { return it.Path == path })
}

func (l *List) add(it Item) {
	i, _ := slices.BinarySearchFunc(l.items, it.Path, func(a Item, p string) int {
		return comparePaths(a.Path, p)
	})
	l.items = slices.Insert(l.items, i, it)
}

// comparePaths orders paths component by component, so "/a/b" sorts
// directly after "/a" and before "/a-b".
func comparePaths(a, b string) int {
	return slices.Compare(components(a), components(b))
}

func components(path string) []string {
	path = strings.Trim(filepath.Clean(path), "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}
