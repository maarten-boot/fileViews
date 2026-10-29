package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/maarten-boot/fileViews/internal/contrast"
	"github.com/maarten-boot/fileViews/internal/listing"
)

func storeIn(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "fileViews", "config.json"))
}

// TC-060
func TestDefaultPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/x")
	if p, err := DefaultPath(); err != nil || p != "/x/fileViews/config.json" {
		t.Errorf("DefaultPath() = %q, %v", p, err)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/u")
	if p, err := DefaultPath(); err != nil || p != "/home/u/.config/fileViews/config.json" {
		t.Errorf("DefaultPath() without XDG = %q, %v", p, err)
	}
}

// TC-062a
func TestDefaults(t *testing.T) {
	c := Default()
	if c.IndentPx != 12 || c.ContrastMin != 4.5 || c.ShowHidden || c.ExactSizes {
		t.Errorf("defaults: %+v", c)
	}
	if c.Window != (Window{1000, 650, 0.25}) {
		t.Errorf("window: %+v", c.Window)
	}
	if c.SortSpec() != listing.DefaultSort {
		t.Errorf("sort: %+v", c.SortSpec())
	}
	if got := c.ColumnLayout().Visible(); !slices.Equal(got, listing.DefaultColumns) {
		t.Errorf("columns: %v", got)
	}
}

// TC-062a
func TestLoadMissing(t *testing.T) {
	s := storeIn(t)
	c, err := s.Load()
	if err != nil {
		t.Fatalf("warning for missing file: %v", err)
	}
	if !reflect.DeepEqual(c, Default()) {
		t.Errorf("got %+v", c)
	}
}

// TC-062b
func TestLoadBroken(t *testing.T) {
	s := storeIn(t)
	os.MkdirAll(filepath.Dir(s.Path), 0o755)
	if err := os.WriteFile(s.Path, []byte("{garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := s.Load()
	if err == nil {
		t.Error("no warning for broken file")
	}
	if !reflect.DeepEqual(c, Default()) {
		t.Errorf("not defaults: %+v", c)
	}
	if data, err := os.ReadFile(s.Path + ".bak"); err != nil || string(data) != "{garbage" {
		t.Errorf(".bak = %q, %v", data, err)
	}
	if _, err := os.Stat(s.Path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("broken file still in place: %v", err)
	}
	// Saving afterwards is allowed: the broken file was moved aside.
	if err := s.Save(c); err != nil {
		t.Errorf("Save after broken load: %v", err)
	}
}

// TC-062c
// An existing file that cannot be read is never overwritten.
func TestLoadUnreadableBlocksSave(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	s := storeIn(t)
	os.MkdirAll(filepath.Dir(s.Path), 0o755)
	if err := os.WriteFile(s.Path, []byte(`{"indentPx": 20}`), 0); err != nil {
		t.Fatal(err)
	}
	c, err := s.Load()
	if err == nil {
		t.Error("no warning for unreadable file")
	}
	if c.IndentPx != DefaultIndentPx {
		t.Errorf("not defaults: %+v", c)
	}
	if err := s.Save(c); !errors.Is(err, ErrNotSaved) {
		t.Errorf("Save = %v, want ErrNotSaved", err)
	}
	os.Chmod(s.Path, 0o644)
	if data, _ := os.ReadFile(s.Path); string(data) != `{"indentPx": 20}` {
		t.Errorf("file was changed: %q", data)
	}
}

// TC-016, TC-061, FR-061: every field survives a round trip.
func TestRoundTrip(t *testing.T) {
	s := storeIn(t)
	want := Default()
	want.Bookmarks = []Bookmark{
		{Path: "/"},
		{Path: "/home/u", Background: "#FFFFFF", Foreground: "#000080", HoverText: "work"},
	}
	want.SetColumnLayout(listing.ColumnLayout{
		Order:  []listing.Column{listing.ColName, listing.ColSize},
		Hidden: map[listing.Column]bool{listing.ColLinks: true, listing.ColGroup: true},
	})
	want.SetSortSpec(listing.SortSpec{Column: listing.ColSize, Descending: true})
	want.IndentPx = 20
	want.ShowHidden = true
	want.ExactSizes = true
	want.ContrastMin = 7
	want.Recent = []string{"/a", "/b"}
	want.LastDir = "/a"
	want.Window = Window{Width: 1200, Height: 800, Split: 0.3}

	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := NewStore(s.Path).Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip:\n got  %+v\n want %+v", got, want)
	}
	if got.SortSpec() != (listing.SortSpec{Column: listing.ColSize, Descending: true}) {
		t.Errorf("sort spec: %+v", got.SortSpec())
	}
	l := got.ColumnLayout()
	if !l.Hidden[listing.ColLinks] || !slices.Equal(l.Order, []listing.Column{listing.ColName, listing.ColSize}) {
		t.Errorf("column layout: %+v", l)
	}
}

// TC-063a
func TestSaveAtomic(t *testing.T) {
	s := storeIn(t)
	for i := range 3 { // creates the directory, then replaces the file
		c := Default()
		c.IndentPx = 10 + i
		if err := s.Save(c); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(filepath.Dir(s.Path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "config.json" {
		t.Errorf("directory contains %v; temporary files left behind?", entries)
	}
	fi, _ := os.Stat(s.Path)
	if fi.Mode().Perm() != 0o644 {
		t.Errorf("mode = %v", fi.Mode().Perm())
	}
	c, _ := s.Load()
	if c.IndentPx != 12 {
		t.Errorf("IndentPx = %d, want last saved value 12", c.IndentPx)
	}
}

// TC-063b: a failed save leaves the existing file intact and no temp file.
func TestSaveFailureKeepsOld(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	s := storeIn(t)
	if err := s.Save(Default()); err != nil {
		t.Fatal(err)
	}
	old, _ := os.ReadFile(s.Path)
	dir := filepath.Dir(s.Path)
	os.Chmod(dir, 0o555)
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	c := Default()
	c.IndentPx = 30
	if err := s.Save(c); err == nil {
		t.Fatal("Save into read-only dir succeeded")
	}
	if now, _ := os.ReadFile(s.Path); string(now) != string(old) {
		t.Error("existing config changed")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("directory contains %v", entries)
	}
}

// TC-064
// Hand-edited or older files with bad values still give a usable config.
func TestNormalize(t *testing.T) {
	s := storeIn(t)
	os.MkdirAll(filepath.Dir(s.Path), 0o755)
	data := `{
		"indentPx": -5,
		"contrastMin": 1,
		"window": {"width": 0, "height": 500, "split": 1.5},
		"sort": {"column": "Bogus", "descending": true},
		"columns": {"order": ["Size", "Bogus", "Name"], "hidden": ["Name", "Owner", "Nope"]},
		"recent": ["/a", "", "/a", "/b"],
		"unknownField": 1
	}`
	if err := os.WriteFile(s.Path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.IndentPx != DefaultIndentPx || c.ContrastMin != contrast.FloorMin {
		t.Errorf("indent %d contrast %v", c.IndentPx, c.ContrastMin)
	}
	if c.Window != (Window{DefaultWidth, DefaultHeight, DefaultSplit}) {
		t.Errorf("window %+v", c.Window)
	}
	if c.SortSpec() != listing.DefaultSort {
		t.Errorf("sort %+v", c.SortSpec())
	}
	if !slices.Equal(c.Recent, []string{"/a", "/b"}) {
		t.Errorf("recent %v", c.Recent)
	}
	want := []listing.Column{listing.ColSize, listing.ColName, listing.ColPerms, listing.ColLinks, listing.ColGroup, listing.ColModified}
	if got := c.ColumnLayout().Visible(); !slices.Equal(got, want) {
		t.Errorf("columns %v, want %v", got, want)
	}

	big := Config{IndentPx: 500, ContrastMin: 50}
	big.Normalize()
	if big.IndentPx != MaxIndentPx || big.ContrastMin != contrast.MaxRatio {
		t.Errorf("upper limits: indent %d contrast %v", big.IndentPx, big.ContrastMin)
	}
}
