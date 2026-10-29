package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/maarten-boot/fileViews/internal/listing"
)

// statusInfo is everything the status bar shows (FR-050).
type statusInfo struct {
	Stats     listing.Stats
	Selected  int   // number of selected entries
	SelSize   int64 // size of the selected entries
	Free      uint64
	FreeKnown bool
	Exact     bool // exact bytes instead of human-readable sizes
}

func (s statusInfo) String() string {
	size := func(n int64) string {
		if s.Exact {
			return strconv.FormatInt(n, 10)
		}
		return listing.HumanSize(n)
	}
	parts := []string{
		plural(s.Stats.Dirs, "directory", "directories"),
		plural(s.Stats.Files, "file", "files"),
		fmt.Sprintf("%d hidden", s.Stats.Hidden),
		size(s.Stats.TotalSize) + " total",
	}
	if s.Selected > 0 {
		parts = append(parts, fmt.Sprintf("%d selected (%s)", s.Selected, size(s.SelSize)))
	}
	if s.FreeKnown {
		parts = append(parts, size(int64(s.Free))+" free")
	}
	return strings.Join(parts, " · ")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}
