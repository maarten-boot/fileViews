package listing

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// mkfile creates name in dir with size bytes and the given mode.
func mkfile(t testing.TB, dir, name string, size int, mode fs.FileMode) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, make([]byte, size), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil { // bypass umask
		t.Fatal(err)
	}
}

func mkdir(t testing.TB, dir, name string) {
	t.Helper()
	if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
}

func readAll(t *testing.T, dir string) []Entry {
	t.Helper()
	entries, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func find(t *testing.T, entries []Entry, name string) Entry {
	t.Helper()
	for _, e := range entries {
		if e.Name == name {
			return e
		}
	}
	t.Fatalf("entry %q not found", name)
	return Entry{}
}

func names(entries []Entry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Name
	}
	return out
}

// TC-020
func TestReadListsFilesAndDirs(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "a.txt", 3, 0o644)
	mkdir(t, dir, "sub")

	entries := readAll(t, dir)
	Sort(entries, DefaultSort)
	if got, want := names(entries), []string{"sub", "a.txt"}; !slices.Equal(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	if e := find(t, entries, "a.txt"); e.Kind != KindFile || e.Size != 3 {
		t.Errorf("a.txt: kind %v size %d", e.Kind, e.Size)
	}
	if e := find(t, entries, "sub"); e.Kind != KindDir || !e.IsDirLike() {
		t.Errorf("sub: kind %v dirlike %v", e.Kind, e.IsDirLike())
	}
}

// TC-011b: an unreadable or missing directory is an error.
func TestReadMissingDir(t *testing.T) {
	if _, err := Read(filepath.Join(t.TempDir(), "nonexistent")); err == nil {
		t.Fatal("expected error")
	}
}

// TC-021a, TC-021b
func TestFilterHidden(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, ".x", 0, 0o644)
	mkfile(t, dir, "y", 0, 0o644)
	all := readAll(t, dir)

	if got := names(Filter(all, false)); !slices.Equal(got, []string{"y"}) {
		t.Errorf("hidden off: %v", got)
	}
	got := names(Filter(all, true))
	slices.Sort(got)
	if !slices.Equal(got, []string{".x", "y"}) {
		t.Errorf("hidden on: %v", got)
	}
}

// TC-022
func TestPermString(t *testing.T) {
	tests := []struct {
		mode fs.FileMode
		want string
	}{
		{0o754, "-rwxr-xr--"},
		{fs.ModeDir | 0o755, "drwxr-xr-x"},
		{fs.ModeSymlink | 0o777, "lrwxrwxrwx"},
		{fs.ModeNamedPipe | 0o644, "prw-r--r--"},
		{fs.ModeSocket | 0o755, "srwxr-xr-x"},
		{fs.ModeDevice | fs.ModeCharDevice | 0o666, "crw-rw-rw-"},
		{fs.ModeDevice | 0o660, "brw-rw----"},
		{fs.ModeSetuid | 0o755, "-rwsr-xr-x"},
		{fs.ModeSetuid | 0o644, "-rwSr--r--"},
		{fs.ModeSetgid | 0o755, "-rwxr-sr-x"},
		{fs.ModeDir | fs.ModeSticky | 0o777, "drwxrwxrwt"},
		{fs.ModeDir | fs.ModeSticky | 0o770, "drwxrwx--T"},
	}
	for _, tt := range tests {
		if got := PermString(tt.mode); got != tt.want {
			t.Errorf("PermString(%v) = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

// TC-022: mode bits survive a Read.
func TestReadPermissions(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "f", 0, 0o754)
	e := find(t, readAll(t, dir), "f")
	if got := FormatField(e, ColPerms, FormatOptions{}); got != "-rwxr-xr--" {
		t.Errorf("perms = %q", got)
	}
	if e.Links != 1 || e.Owner == "" || e.Group == "" {
		t.Errorf("links/owner/group = %d/%q/%q", e.Links, e.Owner, e.Group)
	}
}

// TC-023a, TC-023b, TC-023c
func TestDisplayNameMarkers(t *testing.T) {
	dir := t.TempDir()
	mkdir(t, dir, "sub")
	mkfile(t, dir, "run", 0, 0o755)
	mkfile(t, dir, "a.txt", 0, 0o644)
	if err := os.Symlink("a.txt", filepath.Join(dir, "l")); err != nil {
		t.Fatal(err)
	}
	entries := readAll(t, dir)
	for name, want := range map[string]string{
		"sub":   "sub/",
		"run":   "run*",
		"a.txt": "a.txt",
		"l":     "l@ -> a.txt",
	} {
		if got := DisplayName(find(t, entries, name)); got != want {
			t.Errorf("DisplayName(%s) = %q, want %q", name, got, want)
		}
	}
}

// TC-023d, TC-023e (marker formatting; real FIFO/socket in listing_unix_test.go)
func TestDisplayNameSpecial(t *testing.T) {
	if got := DisplayName(Entry{Name: "p", Kind: KindFIFO}); got != "p|" {
		t.Errorf("fifo: %q", got)
	}
	if got := DisplayName(Entry{Name: "s", Kind: KindSocket}); got != "s=" {
		t.Errorf("socket: %q", got)
	}
}

// TC-025d
// Symlink to a directory counts as a directory for sorting and navigation (FR-025, FR-027).
func TestSymlinkToDir(t *testing.T) {
	dir := t.TempDir()
	mkdir(t, dir, "sub")
	if err := os.Symlink("sub", filepath.Join(dir, "ls")); err != nil {
		t.Fatal(err)
	}
	e := find(t, readAll(t, dir), "ls")
	if e.Kind != KindSymlink || !e.IsDirLike() {
		t.Errorf("kind %v dirlike %v", e.Kind, e.IsDirLike())
	}
}

// TC-024a
func TestColumnOrder(t *testing.T) {
	l := ColumnLayout{Order: []Column{ColName, ColSize, ColPerms}, Hidden: map[Column]bool{
		ColLinks: true, ColOwner: true, ColGroup: true, ColModified: true,
	}}
	cols := l.Visible()
	if want := []Column{ColName, ColSize, ColPerms}; !slices.Equal(cols, want) {
		t.Fatalf("Visible() = %v, want %v", cols, want)
	}
	e := Entry{Name: "f", Kind: KindFile, Mode: 0o644, Size: 10}
	if got, want := FormatRow(e, cols, FormatOptions{}), []string{"f", "10", "-rw-r--r--"}; !slices.Equal(got, want) {
		t.Errorf("FormatRow = %v, want %v", got, want)
	}
}

// TC-024b, TC-024c: Name is always shown; stale or bad config is cleaned up.
func TestColumnLayoutNormalises(t *testing.T) {
	tests := []struct {
		name   string
		layout ColumnLayout
		want   []Column
	}{
		{"empty is default", ColumnLayout{}, DefaultColumns},
		{"name cannot be hidden", ColumnLayout{Order: []Column{ColSize, ColName}, Hidden: map[Column]bool{ColName: true}},
			[]Column{ColSize, ColName, ColPerms, ColLinks, ColOwner, ColGroup, ColModified}},
		{"duplicates and unknown dropped", ColumnLayout{Order: []Column{ColSize, ColSize, Column(99), Column(-1)}},
			[]Column{ColSize, ColPerms, ColLinks, ColOwner, ColGroup, ColModified, ColName}},
	}
	for _, tt := range tests {
		if got := tt.layout.Visible(); !slices.Equal(got, tt.want) {
			t.Errorf("%s: Visible() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestParseColumn(t *testing.T) {
	for _, c := range DefaultColumns {
		got, ok := ParseColumn(c.String())
		if !ok || got != c {
			t.Errorf("ParseColumn(%q) = %v, %v", c.String(), got, ok)
		}
	}
	if _, ok := ParseColumn("Bogus"); ok {
		t.Error("ParseColumn(Bogus) ok")
	}
}

// TC-025a
func TestDefaultSort(t *testing.T) {
	entries := []Entry{
		{Name: "b"},
		{Name: "A", Kind: KindDir, TargetIsDir: true},
		{Name: "a"},
		{Name: "B"},
	}
	Sort(entries, DefaultSort)
	if got, want := names(entries), []string{"A", "a", "B", "b"}; !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

// TC-025b: directories stay first when descending.
func TestSortBySizeDescending(t *testing.T) {
	entries := []Entry{
		{Name: "one", Size: 1},
		{Name: "three", Size: 3},
		{Name: "d", Kind: KindDir, TargetIsDir: true, Size: 4096},
		{Name: "two", Size: 2},
	}
	Sort(entries, SortSpec{Column: ColSize, Descending: true})
	if got, want := names(entries), []string{"d", "three", "two", "one"}; !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

// TC-025c
func TestSortTiesByName(t *testing.T) {
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	entries := []Entry{{Name: "c", ModTime: ts}, {Name: "a", ModTime: ts}, {Name: "b", ModTime: ts.Add(-time.Hour)}}
	Sort(entries, SortSpec{Column: ColModified})
	if got, want := names(entries), []string{"b", "a", "c"}; !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

// TC-026a, TC-026b
func TestSizeFormat(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{1023, "1023"},
		{1024, "1.0K"},
		{4300, "4.2K"},
		{10 * 1024, "10K"},
		{1536 * 1024, "1.5M"},
		{5 << 30, "5.0G"},
	}
	for _, tt := range tests {
		if got := HumanSize(tt.n); got != tt.want {
			t.Errorf("HumanSize(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
	e := Entry{Kind: KindFile, Mode: 0o644, Size: 4300}
	if got := FormatField(e, ColSize, FormatOptions{ExactSizes: true}); got != "4300" {
		t.Errorf("exact = %q", got)
	}
	if got := FormatField(e, ColSize, FormatOptions{}); got != "4.2K" {
		t.Errorf("human = %q", got)
	}
}

// TC-026c
func TestModifiedFormat(t *testing.T) {
	e := Entry{Kind: KindFile, Mode: 0o644, ModTime: time.Date(2026, 10, 29, 11, 4, 59, 0, time.Local)}
	if got := FormatField(e, ColModified, FormatOptions{}); got != "2026-10-29 11:04" {
		t.Errorf("modified = %q", got)
	}
}

// TC-029a
func TestBrokenSymlink(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "ok", 0, 0o644)
	if err := os.Symlink("missing", filepath.Join(dir, "l")); err != nil {
		t.Fatal(err)
	}
	entries := readAll(t, dir)
	if len(entries) != 2 {
		t.Fatalf("got %d entries", len(entries))
	}
	e := find(t, entries, "l")
	if e.Err != nil || e.IsDirLike() || DisplayName(e) != "l@ -> missing" {
		t.Errorf("broken link: err %v dirlike %v name %q", e.Err, e.IsDirLike(), DisplayName(e))
	}
}

// TC-029c
func TestUnreadableFieldsShowQuestionMark(t *testing.T) {
	e := Entry{Name: "x", Err: fs.ErrPermission}
	if got := FormatRow(e, DefaultColumns, FormatOptions{}); got[0] != "?" || got[6] != "x" {
		t.Errorf("row = %v", got)
	}
}

// TC-050a, TC-050b
func TestComputeStats(t *testing.T) {
	all := []Entry{
		{Name: "d1", Kind: KindDir, TargetIsDir: true, Size: 4096},
		{Name: ".d2", Kind: KindDir, TargetIsDir: true, Size: 4096},
		{Name: "f1", Size: 100},
		{Name: "f2", Size: 200},
		{Name: ".f3", Size: 50},
	}
	if got, want := ComputeStats(all, true), (Stats{Dirs: 2, Files: 3, Hidden: 2, TotalSize: 350}); got != want {
		t.Errorf("show hidden: %+v, want %+v", got, want)
	}
	if got, want := ComputeStats(all, false), (Stats{Dirs: 1, Files: 2, Hidden: 2, TotalSize: 300}); got != want {
		t.Errorf("hide hidden: %+v, want %+v", got, want)
	}
}

// TC-050c
func TestSelectionSize(t *testing.T) {
	sel := []Entry{{Name: "f", Size: 100}, {Name: "d", Kind: KindDir, TargetIsDir: true, Size: 4096}}
	if got := TotalSize(sel); got != 100 {
		t.Errorf("TotalSize = %d", got)
	}
}

// TC-N01: 10,000 entries must list (read + sort + format) well under 1 s.
func BenchmarkListLargeDir(b *testing.B) {
	dir := b.TempDir()
	for i := range 10000 {
		mkfile(b, dir, fmt.Sprintf("file%05d", i), i%4096, 0o644)
	}
	b.ResetTimer()
	for b.Loop() {
		entries, err := Read(dir)
		if err != nil {
			b.Fatal(err)
		}
		Sort(entries, DefaultSort)
		for _, e := range entries {
			FormatRow(e, DefaultColumns, FormatOptions{})
		}
	}
}
