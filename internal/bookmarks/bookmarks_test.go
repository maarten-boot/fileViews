package bookmarks

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/maarten-boot/fileViews/internal/config"
	"github.com/maarten-boot/fileViews/internal/contrast"
)

func paths(l *List) []string {
	var out []string
	for _, it := range l.Items() {
		out = append(out, it.Path)
	}
	return out
}

func mustGet(t *testing.T, l *List, path string) Item {
	t.Helper()
	it, ok := l.Get(path)
	if !ok {
		t.Fatalf("%s not in list", path)
	}
	return it
}

// TC-041a
func TestDefaultItems(t *testing.T) {
	l := New("/home/mboot", nil)
	items := l.Items()
	if len(items) != 2 {
		t.Fatalf("items = %v", items)
	}
	if items[0].Label() != "/" || !items[0].Default {
		t.Errorf("first item %+v", items[0])
	}
	if items[1].Label() != "mboot" || items[1].Path != "/home/mboot" || !items[1].Default {
		t.Errorf("second item %+v", items[1])
	}
}

// TC-041c
func TestHomeIsRoot(t *testing.T) {
	if got := paths(New("/", nil)); !slices.Equal(got, []string{"/"}) {
		t.Errorf("paths = %v", got)
	}
}

// TC-041b, TC-042a
func TestAddRemove(t *testing.T) {
	l := New("/home/mboot", nil)
	if err := l.Remove("/"); !errors.Is(err, ErrIsDefault) {
		t.Errorf("Remove(/) = %v", err)
	}
	if err := l.Remove("/home/mboot"); !errors.Is(err, ErrIsDefault) {
		t.Errorf("Remove(home) = %v", err)
	}
	if err := l.Add("/tmp"); err != nil {
		t.Fatal(err)
	}
	if err := l.Add("/tmp/"); !errors.Is(err, ErrExists) {
		t.Errorf("duplicate Add = %v", err)
	}
	if err := l.Add("tmp"); !errors.Is(err, ErrNotAbsolute) {
		t.Errorf("relative Add = %v", err)
	}
	if err := l.Remove("/tmp"); err != nil {
		t.Fatal(err)
	}
	if err := l.Remove("/tmp"); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Remove = %v", err)
	}
	if got := paths(l); !slices.Equal(got, []string{"/", "/home/mboot"}) {
		t.Errorf("paths = %v", got)
	}
}

// TC-040
// Items sort component-wise so the indent reads like a tree.
// TC-040: the user's order; Add appends.
func TestOrder(t *testing.T) {
	l := New("/home/mboot", nil)
	for _, p := range []string{"/home/mboot-old", "/home/mboot/projects", "/etc"} {
		if err := l.Add(p); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"/", "/home/mboot", "/home/mboot-old", "/home/mboot/projects", "/etc"}
	if got := paths(l); !slices.Equal(got, want) {
		t.Errorf("paths = %v, want %v", got, want)
	}
}

// TC-048a: Insert at a position, Move up and down, already bookmarked is refused.
func TestInsertMove(t *testing.T) {
	l := New("/h", nil) // /, /h
	if err := l.Insert("/a", 1); err != nil {
		t.Fatal(err)
	}
	l.Insert("/b", 99) // clamped to the end
	l.Insert("/c", -5) // clamped to the start
	if got, want := paths(l), []string{"/c", "/", "/a", "/h", "/b"}; !slices.Equal(got, want) {
		t.Fatalf("after insert %v, want %v", got, want)
	}
	if err := l.Insert("/a/", 0); !errors.Is(err, ErrExists) || l.Index("/a") != 2 {
		t.Errorf("duplicate insert: %v, index %d", err, l.Index("/a"))
	}
	l.Move("/c", 4)  // to the end
	l.Move("/h", 0)  // default items move too
	l.Move("/b", -1) // clamped
	if got, want := paths(l), []string{"/b", "/h", "/", "/a", "/c"}; !slices.Equal(got, want) {
		t.Errorf("after move %v, want %v", got, want)
	}
	if err := l.Move("/nope", 0); !errors.Is(err, ErrNotFound) {
		t.Errorf("move unknown: %v", err)
	}
	if it, _ := l.Get("/h"); !it.Default {
		t.Error("moved default lost its default flag")
	}
}

// TC-044a, TC-044b
func TestDepthAndIndent(t *testing.T) {
	for path, want := range map[string]int{"/": 0, "/home": 1, "/home/mboot": 2, "/a/b/c/": 3} {
		it := Item{Path: path}
		if got := it.Depth(); got != want {
			t.Errorf("Depth(%s) = %d, want %d", path, got, want)
		}
	}
	if got := (Item{Path: "/home/mboot"}).Indent(12); got != 24 {
		t.Errorf("Indent = %d, want 24", got)
	}
}

// TC-046a, TC-046b, TC-046c
func TestHoverText(t *testing.T) {
	l := New("/home/mboot", nil)
	if got := mustGet(t, l, "/home/mboot").Tooltip(); got != "/home/mboot" {
		t.Errorf("default tooltip = %q", got)
	}
	l.SetHoverText("/home/mboot", "work")
	if got := mustGet(t, l, "/home/mboot").Tooltip(); got != "work" {
		t.Errorf("tooltip = %q", got)
	}
	l.SetHoverText("/home/mboot", "   ")
	if got := mustGet(t, l, "/home/mboot").Tooltip(); got != "/home/mboot" {
		t.Errorf("blank text did not reset: %q", got)
	}
	if err := l.SetHoverText("/nope", "x"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown path: %v", err)
	}
}

// TC-045a, TC-075b: accepted and refused colour combinations.
func TestSetColours(t *testing.T) {
	l := New("/home/mboot", nil)
	if err := l.SetColours("/", "#ffffff", "#000080", contrast.DefaultMin); err != nil {
		t.Fatal(err)
	}
	if it := mustGet(t, l, "/"); it.Background != "#FFFFFF" || it.Foreground != "#000080" {
		t.Errorf("stored %+v", it)
	}

	// Yellow text on the default white background is refused; nothing changes.
	err := l.SetColours("/home/mboot", "", "#FFFF00", contrast.DefaultMin)
	var lc *LowContrastError
	if !errors.As(err, &lc) || lc.Suggested != "#000000" || lc.Ratio > 1.1 {
		t.Fatalf("err = %v", err)
	}
	if it := mustGet(t, l, "/home/mboot"); it.Foreground != "" {
		t.Errorf("refused colour stored: %+v", it)
	}
	// Dark background: the suggestion is white.
	err = l.SetColours("/home/mboot", "#000080", "", contrast.DefaultMin) // black text on navy
	if !errors.As(err, &lc) || lc.Suggested != "#FFFFFF" {
		t.Errorf("navy background: %v", err)
	}
	if err := l.SetColours("/", "#GGGGGG", "", contrast.DefaultMin); err == nil {
		t.Error("invalid hex accepted")
	}
	if err := l.ResetColours("/"); err != nil {
		t.Fatal(err)
	}
	if it := mustGet(t, l, "/"); it.Background != "" || it.Foreground != "" {
		t.Errorf("not reset: %+v", it)
	}
}

// TC-070a, TC-072, TC-076: drawing colours.
func TestStyleColours(t *testing.T) {
	st := (Item{Path: "/a"}).Style(contrast.DefaultMin, false)
	if st.Background != contrast.White || st.Foreground != contrast.Black || st.Italic || st.Label != "a" {
		t.Errorf("default style %+v", st)
	}
	// A saved combination that fails is drawn with black text; the item keeps its colours.
	it := Item{Path: "/a", Foreground: "#FFFF00"}
	if st := it.Style(contrast.DefaultMin, false); st.Foreground != contrast.Black {
		t.Errorf("failing fg drawn as %s", contrast.Hex(st.Foreground))
	}
	if it.Foreground != "#FFFF00" {
		t.Error("saved colour changed")
	}
	// Raising the minimum makes a previously fine colour fall back.
	it = Item{Path: "/a", Foreground: "#767676"}
	if st := it.Style(4.5, false); contrast.Hex(st.Foreground) != "#767676" {
		t.Errorf("at 4.5: %s", contrast.Hex(st.Foreground))
	}
	if st := it.Style(7, false); st.Foreground != contrast.Black {
		t.Errorf("at 7: %s", contrast.Hex(st.Foreground))
	}
	// Invalid stored colours fall back to the defaults.
	if st := (Item{Path: "/a", Background: "bad"}).Style(4.5, false); st.Background != contrast.White {
		t.Errorf("invalid bg drawn as %s", contrast.Hex(st.Background))
	}
}

// TC-047b, FR-071
func TestStyleMissing(t *testing.T) {
	tests := []struct {
		name     string
		item     Item
		wantText string
	}{
		{"default white: gray", Item{Path: "/x/gone"}, "#767676"},
		{"black background: gray", Item{Path: "/x/gone", Background: "#000000", Foreground: "#FFFFFF"}, "#757575"},
		{"own background: own colour", Item{Path: "/x/gone", Background: "#FFFFE0", Foreground: "#000080"}, "#000080"},
	}
	for _, tt := range tests {
		st := tt.item.Style(contrast.DefaultMin, true)
		if st.Label != "gone (missing)" || !st.Italic {
			t.Errorf("%s: label %q italic %v", tt.name, st.Label, st.Italic)
		}
		if got := contrast.Hex(st.Foreground); got != tt.wantText {
			t.Errorf("%s: text %s, want %s", tt.name, got, tt.wantText)
		}
		if !contrast.Meets(st.Foreground, st.Background, contrast.DefaultMin) {
			t.Errorf("%s: fails the contrast minimum", tt.name)
		}
	}
}

// TC-047a
func TestMissing(t *testing.T) {
	dir := t.TempDir()
	if (Item{Path: dir}).Missing() {
		t.Error("existing dir reported missing")
	}
	if !(Item{Path: filepath.Join(dir, "gone")}).Missing() {
		t.Error("missing dir not reported")
	}
	f := filepath.Join(dir, "file")
	os.WriteFile(f, nil, 0o644)
	if !(Item{Path: f}).Missing() {
		t.Error("file reported as directory")
	}
}

// TC-045b, TC-046b: settings survive a save/load through the config.
func TestSavedRoundTrip(t *testing.T) {
	l := New("/home/mboot", nil)
	l.Add("/tmp")
	l.SetColours("/tmp", "#FFFFE0", "#000080", contrast.DefaultMin)
	l.SetHoverText("/home/mboot", "work")

	store := config.NewStore(filepath.Join(t.TempDir(), "config.json"))
	cfg := config.Default()
	cfg.Bookmarks = l.Saved()
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	l2 := New("/home/mboot", loaded.Bookmarks)
	if !slices.Equal(l2.Items(), l.Items()) {
		t.Errorf("after reload:\n got  %+v\n want %+v", l2.Items(), l.Items())
	}
}

// TC-042b
// Saved data from a hand-edited or older config is cleaned up.
func TestNewFromSavedCleansUp(t *testing.T) {
	saved := []config.Bookmark{
		{Path: "relative"},
		{Path: ""},
		{Path: "/tmp/"},
		{Path: "/tmp", HoverText: "second"},
		{Path: "/", Background: "#000000", Foreground: "#ffffff"},
		{Path: "/opt", Foreground: "not-a-colour"},
	}
	l := New("/home/mboot", saved)
	// Saved order is kept; "/" was saved, home was not and goes right after "/".
	if got, want := paths(l), []string{"/tmp", "/", "/home/mboot", "/opt"}; !slices.Equal(got, want) {
		t.Errorf("paths = %v, want %v", got, want)
	}
	// The first entry wins, even when it has no settings.
	if it := mustGet(t, l, "/tmp"); it.HoverText != "" || it.Default {
		t.Errorf("/tmp = %+v", it)
	}
	if it := mustGet(t, l, "/"); !it.Default || it.Background != "#000000" || it.Foreground != "#FFFFFF" {
		t.Errorf("/ = %+v", it)
	}
	if it := mustGet(t, l, "/opt"); it.Foreground != "" {
		t.Errorf("/opt = %+v", it)
	}
	// A saved home from another account is just a user bookmark now.
	l = New("/home/other", []config.Bookmark{{Path: "/home/mboot"}})
	if it := mustGet(t, l, "/home/mboot"); it.Default {
		t.Error("old home kept as default")
	}
}
