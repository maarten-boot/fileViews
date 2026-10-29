//go:build unix

package listing

import (
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// TC-023d
func TestFIFO(t *testing.T) {
	dir := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(dir, "p"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := find(t, readAll(t, dir), "p")
	if e.Kind != KindFIFO || DisplayName(e) != "p|" || PermString(e.Mode) != "prw-r--r--" {
		t.Errorf("fifo: kind %v name %q perms %q", e.Kind, DisplayName(e), PermString(e.Mode))
	}
}

// TC-023e
func TestSocket(t *testing.T) {
	dir := t.TempDir()
	l, err := net.Listen("unix", filepath.Join(dir, "s"))
	if err != nil {
		t.Skip("cannot create unix socket:", err)
	}
	defer l.Close()
	e := find(t, readAll(t, dir), "s")
	if e.Kind != KindSocket || DisplayName(e) != "s=" {
		t.Errorf("socket: kind %v name %q", e.Kind, DisplayName(e))
	}
}

// TC-029b: an unreadable subdirectory is still listed.
func TestUnreadableSubdir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	dir := t.TempDir()
	mkdir(t, dir, "sub")
	sub := filepath.Join(dir, "sub")
	if err := os.Chmod(sub, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(sub, 0o755) })

	e := find(t, readAll(t, dir), "sub")
	if DisplayName(e) != "sub/" || PermString(e.Mode) != "d---------" {
		t.Errorf("sub: name %q perms %q", DisplayName(e), PermString(e.Mode))
	}
	if _, err := Read(sub); err == nil {
		t.Error("reading unreadable dir: expected error")
	}
}

func TestUnknownIDsAreNumeric(t *testing.T) {
	const id = 4000000000 // not a real uid/gid
	if got := userName(id); got != "4000000000" {
		t.Errorf("userName = %q", got)
	}
	if got := groupName(id); got != "4000000000" {
		t.Errorf("groupName = %q", got)
	}
}

func TestFreeSpace(t *testing.T) {
	n, err := FreeSpace(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Log("free space reported as 0 (full filesystem?)")
	}
	if _, err := FreeSpace(filepath.Join(t.TempDir(), "nonexistent")); err == nil {
		t.Error("expected error for missing path")
	}
}
