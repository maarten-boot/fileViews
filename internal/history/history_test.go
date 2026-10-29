package history

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TC-012a
func TestBackForward(t *testing.T) {
	var h History
	for _, d := range []string{"A", "B", "C"} {
		h.Visit(d)
	}
	h.Back()
	h.Back()
	if got, _ := h.Forward(); got != "B" || h.Current() != "B" {
		t.Errorf("current = %q", h.Current())
	}
	if !h.CanBack() || !h.CanForward() {
		t.Errorf("CanBack %v CanForward %v, want both true", h.CanBack(), h.CanForward())
	}
}

// TC-012b
func TestVisitClearsForward(t *testing.T) {
	var h History
	h.Visit("A")
	h.Visit("B")
	h.Back()
	h.Visit("C")
	if h.CanForward() {
		t.Error("forward not cleared")
	}
	if got, _ := h.Back(); got != "A" {
		t.Errorf("back = %q, want A", got)
	}
}

// TC-012c
func TestHistoryEdges(t *testing.T) {
	var h History
	if h.Current() != "" || h.CanBack() || h.CanForward() {
		t.Error("empty history not empty")
	}
	if _, ok := h.Back(); ok {
		t.Error("Back on empty history")
	}
	h.Visit("A")
	h.Visit("A") // revisiting current is a no-op
	if h.CanBack() {
		t.Error("duplicate visit added an entry")
	}
	if _, ok := h.Forward(); ok {
		t.Error("Forward at end")
	}
}

// TC-015a
func TestRecentNoDuplicates(t *testing.T) {
	r := NewRecent(0, nil)
	for _, d := range []string{"A", "B", "A"} {
		r.Add(d)
	}
	if got, want := r.Items(), []string{"A", "B"}; !slices.Equal(got, want) {
		t.Errorf("recent = %v, want %v", got, want)
	}
}

// TC-015b
func TestRecentMax(t *testing.T) {
	r := NewRecent(0, nil)
	for i := range 25 {
		r.Add(fmt.Sprintf("d%02d", i))
	}
	got := r.Items()
	if len(got) != DefaultRecentMax || got[0] != "d24" || got[19] != "d05" {
		t.Errorf("recent = %v", got)
	}
}

// TC-016: a list loaded from config is cleaned up.
func TestRecentFromConfig(t *testing.T) {
	r := NewRecent(3, []string{"A", "", "B", "A", "C", "D"})
	if got, want := r.Items(), []string{"A", "B", "C"}; !slices.Equal(got, want) {
		t.Errorf("recent = %v, want %v", got, want)
	}
	items := r.Items()
	items[0] = "changed"
	if r.Items()[0] != "A" {
		t.Error("Items does not return a copy")
	}
}

// TC-013
func TestParent(t *testing.T) {
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"/", "", false},
		{"", "", false},
		{"/home", "/", true},
		{"/home/mboot", "/home", true},
		{"/home/mboot/", "/home", true},
	}
	for _, tt := range tests {
		got, ok := Parent(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("Parent(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

// tree creates root/a/b and root/file and returns root.
func tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func mustGo(t *testing.T, n *Navigator, path string) string {
	t.Helper()
	dir, err := n.Go(path)
	if err != nil {
		t.Fatalf("Go(%q): %v", path, err)
	}
	return dir
}

// TC-011b
func TestGoInvalidKeepsCurrent(t *testing.T) {
	root := tree(t)
	n := NewNavigator(nil, root)
	mustGo(t, n, root)
	for _, bad := range []string{
		filepath.Join(root, "nonexistent"),
		filepath.Join(root, "file"), // not a directory
		"",
	} {
		if _, err := n.Go(bad); err == nil {
			t.Errorf("Go(%q) succeeded", bad)
		}
		if n.Current() != root {
			t.Errorf("after Go(%q): current = %q", bad, n.Current())
		}
	}
	if recent := n.Recent(); len(recent) != 1 {
		t.Errorf("failed navigation changed recent: %v", recent)
	}
}

// TC-011c
func TestGoUnreadableDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	root := tree(t)
	locked := filepath.Join(root, "a")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	n := NewNavigator(nil, root)
	mustGo(t, n, root)
	if _, err := n.Go(locked); !errors.Is(err, fs.ErrPermission) {
		t.Errorf("err = %v, want permission error", err)
	}
	if n.Current() != root {
		t.Errorf("current = %q", n.Current())
	}
}

// TC-011d
func TestGoResolvesPaths(t *testing.T) {
	root := tree(t)
	n := NewNavigator(nil, root)
	mustGo(t, n, root)
	tests := []struct{ in, want string }{
		{"a", filepath.Join(root, "a")},                   // relative
		{"b/..", filepath.Join(root, "a")},                // cleaned
		{"~", root},                                       // home
		{"~/a/b", filepath.Join(root, "a", "b")},          // home-relative
		{"  " + root + "/a/  ", filepath.Join(root, "a")}, // trimmed
	}
	for _, tt := range tests {
		if got := mustGo(t, n, tt.in); got != tt.want {
			t.Errorf("Go(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TC-011e
func TestRelativeWithoutCurrent(t *testing.T) {
	if _, err := NewNavigator(nil, "/").Go("a"); err == nil {
		t.Error("relative path accepted without a current directory")
	}
}

// TC-012a, TC-015c: back/forward through the Navigator, and recent updates on back/forward.
func TestNavigatorBackForward(t *testing.T) {
	root := tree(t)
	a, b := filepath.Join(root, "a"), filepath.Join(root, "a", "b")
	n := NewNavigator(nil, root)
	mustGo(t, n, root)
	mustGo(t, n, a)
	mustGo(t, n, b)

	if dir, err := n.Back(); err != nil || dir != a {
		t.Fatalf("Back = %q, %v", dir, err)
	}
	if got := n.Recent()[0]; got != a {
		t.Errorf("recent[0] after Back = %q, want %q", got, a)
	}
	if dir, err := n.Forward(); err != nil || dir != b {
		t.Fatalf("Forward = %q, %v", dir, err)
	}
	if _, err := n.Forward(); !errors.Is(err, ErrNoForward) {
		t.Errorf("Forward at end: %v", err)
	}
}

// TC-012d
// Back to a directory that was removed: error, position unchanged.
func TestBackToRemovedDir(t *testing.T) {
	root := tree(t)
	b := filepath.Join(root, "a", "b")
	n := NewNavigator(nil, root)
	mustGo(t, n, b)
	mustGo(t, n, root)
	if err := os.Remove(b); err != nil {
		t.Fatal(err)
	}
	if _, err := n.Back(); err == nil {
		t.Fatal("Back to removed dir succeeded")
	}
	if n.Current() != root || !n.CanBack() {
		t.Errorf("current = %q, CanBack = %v", n.Current(), n.CanBack())
	}
}

// TC-013, TC-014
func TestUpAndHome(t *testing.T) {
	root := tree(t)
	n := NewNavigator(nil, root)
	mustGo(t, n, filepath.Join(root, "a", "b"))
	if dir, err := n.Up(); err != nil || dir != filepath.Join(root, "a") {
		t.Errorf("Up = %q, %v", dir, err)
	}
	if dir, err := n.Home(); err != nil || dir != root {
		t.Errorf("Home = %q, %v", dir, err)
	}
	mustGo(t, n, "/")
	if n.CanUp() {
		t.Error("CanUp at /")
	}
	if _, err := n.Up(); !errors.Is(err, ErrAtRoot) {
		t.Errorf("Up at / = %v", err)
	}
	if n.Current() != "/" {
		t.Errorf("current = %q", n.Current())
	}
}

// TC-012e: Target does not move; Commit moves; a Back whose target is no
// longer the previous entry becomes a plain visit.
func TestTargetCommit(t *testing.T) {
	root := tree(t)
	a, b := filepath.Join(root, "a"), filepath.Join(root, "a", "b")
	n := NewNavigator(nil, root)
	mustGo(t, n, a)
	mustGo(t, n, b)

	dir, err := n.Target(MoveBack, "")
	if err != nil || dir != a || n.Current() != b {
		t.Fatalf("Target(Back) = %q, %v; current %q", dir, err, n.Current())
	}
	for mv, want := range map[Move]string{MoveUp: a, MoveHome: root, MoveGo: filepath.Join(b, "x")} {
		if got, err := n.Target(mv, "x"); err != nil || got != want {
			t.Errorf("Target(%v) = %q, %v; want %q", mv, got, err, want)
		}
	}
	if _, err := n.Target(MoveForward, ""); !errors.Is(err, ErrNoForward) {
		t.Errorf("Target(Forward) err = %v", err)
	}
	n.Commit(MoveBack, a)
	if n.Current() != a || !n.CanForward() {
		t.Errorf("after Commit(Back): %q forward %v", n.Current(), n.CanForward())
	}
	if n.Recent()[0] != a {
		t.Errorf("recent %v", n.Recent())
	}
	// Stale Back: history moved on before the commit.
	n.Commit(MoveGo, root)
	n.Commit(MoveBack, b) // b is not the previous entry (a is): plain visit
	if n.Current() != b || n.CanForward() {
		t.Errorf("stale back: current %q forward %v", n.Current(), n.CanForward())
	}
	if back, _ := n.Target(MoveBack, ""); back != root {
		t.Errorf("previous entry %q, want %q", back, root)
	}
}
