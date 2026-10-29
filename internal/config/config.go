package config

import (
	"github.com/maarten-boot/fileViews/internal/contrast"
	"github.com/maarten-boot/fileViews/internal/history"
	"github.com/maarten-boot/fileViews/internal/listing"
)

// CurrentVersion is the config file format version written by Save.
const CurrentVersion = 1

// Defaults for settings whose zero value is not a sensible default.
const (
	DefaultIndentPx = 12
	MaxIndentPx     = 64
	DefaultWidth    = 1000
	DefaultHeight   = 650
	DefaultSplit    = 0.25
)

// Config is everything that is kept across sessions (FR-061).
type Config struct {
	Version     int        `json:"version"`
	Bookmarks   []Bookmark `json:"bookmarks,omitempty"`
	Columns     Columns    `json:"columns"`
	Sort        Sort       `json:"sort"`
	IndentPx    int        `json:"indentPx"`
	ShowHidden  bool       `json:"showHidden"`
	ExactSizes  bool       `json:"exactSizes"`
	ContrastMin float64    `json:"contrastMin"`
	Recent      []string   `json:"recent,omitempty"`
	LastDir     string     `json:"lastDir,omitempty"`
	Window      Window     `json:"window"`
}

// Bookmark is one side pane item as stored. Colours are "#RRGGBB" or empty
// for the default; an empty HoverText means the default (the full path).
type Bookmark struct {
	Path       string `json:"path"`
	Background string `json:"background,omitempty"`
	Foreground string `json:"foreground,omitempty"`
	HoverText  string `json:"hoverText,omitempty"`
}

// Columns stores the column layout by column name (listing.Column.String).
type Columns struct {
	Order  []string `json:"order,omitempty"`
	Hidden []string `json:"hidden,omitempty"`
}

// Sort stores the sort column by name and the direction.
type Sort struct {
	Column     string `json:"column,omitempty"`
	Descending bool   `json:"descending,omitempty"`
}

// Window stores the window size and the left/right pane split (0..1).
type Window struct {
	Width  float32 `json:"width"`
	Height float32 `json:"height"`
	Split  float64 `json:"split"`
}

// Default returns the configuration used when there is no config file.
func Default() Config {
	var c Config
	c.Normalize()
	return c
}

// Normalize replaces missing or out-of-range values with defaults, so a
// hand-edited or older config file still gives a usable configuration.
func (c *Config) Normalize() {
	c.Version = CurrentVersion
	if c.IndentPx <= 0 {
		c.IndentPx = DefaultIndentPx
	}
	c.IndentPx = min(c.IndentPx, MaxIndentPx)
	if c.ContrastMin == 0 {
		c.ContrastMin = contrast.DefaultMin
	}
	c.ContrastMin = contrast.ClampMin(c.ContrastMin)
	if c.Window.Width <= 0 || c.Window.Height <= 0 {
		c.Window.Width, c.Window.Height = DefaultWidth, DefaultHeight
	}
	if c.Window.Split <= 0 || c.Window.Split >= 1 {
		c.Window.Split = DefaultSplit
	}
	if _, ok := listing.ParseColumn(c.Sort.Column); !ok {
		c.Sort = Sort{Column: listing.ColName.String()}
	}
	c.Recent = history.NewRecent(history.DefaultRecentMax, c.Recent).Items()
}

// ColumnLayout converts the stored columns; unknown names are ignored.
func (c Config) ColumnLayout() listing.ColumnLayout {
	l := listing.ColumnLayout{Hidden: map[listing.Column]bool{}}
	for _, name := range c.Columns.Order {
		if col, ok := listing.ParseColumn(name); ok {
			l.Order = append(l.Order, col)
		}
	}
	for _, name := range c.Columns.Hidden {
		if col, ok := listing.ParseColumn(name); ok {
			l.Hidden[col] = true
		}
	}
	return l
}

// SetColumnLayout stores l by column name.
func (c *Config) SetColumnLayout(l listing.ColumnLayout) {
	c.Columns = Columns{}
	for _, col := range l.Order {
		c.Columns.Order = append(c.Columns.Order, col.String())
	}
	for _, col := range listing.DefaultColumns {
		if l.Hidden[col] {
			c.Columns.Hidden = append(c.Columns.Hidden, col.String())
		}
	}
}

// SortSpec converts the stored sort settings.
func (c Config) SortSpec() listing.SortSpec {
	col, ok := listing.ParseColumn(c.Sort.Column)
	if !ok {
		return listing.DefaultSort
	}
	return listing.SortSpec{Column: col, Descending: c.Sort.Descending}
}

// SetSortSpec stores s.
func (c *Config) SetSortSpec(s listing.SortSpec) {
	c.Sort = Sort{Column: s.Column.String(), Descending: s.Descending}
}
