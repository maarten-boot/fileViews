// Package arch holds architecture checks for the module.
package arch

import (
	"os/exec"
	"strings"
	"testing"
)

// TC-N03: core packages must not depend on the GUI toolkit (NFR-003).
func TestCoreHasNoGUIDeps(t *testing.T) {
	core := []string{"listing", "history", "bookmarks", "config", "contrast"}
	for _, pkg := range core {
		path := "github.com/maarten-boot/fileViews/internal/" + pkg
		out, err := exec.Command("go", "list", "-deps", path).Output()
		if err != nil {
			t.Fatalf("go list %s: %v", path, err)
		}
		for _, dep := range strings.Fields(string(out)) {
			if strings.HasPrefix(dep, "fyne.io/") || strings.HasSuffix(dep, "/internal/ui") {
				t.Errorf("%s depends on %s", pkg, dep)
			}
		}
	}
}
