package listing

import (
	"cmp"
	"slices"
	"strings"
)

// SortSpec is the sort column and direction (FR-025).
type SortSpec struct {
	Column     Column
	Descending bool
}

// DefaultSort sorts by name, ascending.
var DefaultSort = SortSpec{Column: ColName}

// Sort sorts entries in place. Directories (and symlinks to directories)
// always come first; within each group entries are ordered by the sort
// column, with ties broken by name.
func Sort(entries []Entry, s SortSpec) {
	slices.SortStableFunc(entries, func(a, b Entry) int {
		if a.IsDirLike() != b.IsDirLike() {
			if a.IsDirLike() {
				return -1
			}
			return 1
		}
		c := compareBy(a, b, s.Column)
		if c == 0 && s.Column != ColName {
			c = compareNames(a.Name, b.Name)
		}
		if s.Descending {
			c = -c
		}
		return c
	})
}

func compareBy(a, b Entry, col Column) int {
	switch col {
	case ColPerms:
		return strings.Compare(PermString(a.Mode), PermString(b.Mode))
	case ColLinks:
		return cmp.Compare(a.Links, b.Links)
	case ColOwner:
		return strings.Compare(a.Owner, b.Owner)
	case ColGroup:
		return strings.Compare(a.Group, b.Group)
	case ColSize:
		return cmp.Compare(a.Size, b.Size)
	case ColModified:
		return a.ModTime.Compare(b.ModTime)
	default:
		return compareNames(a.Name, b.Name)
	}
}

// compareNames compares case-insensitively, falling back to a byte compare
// so the order is total.
func compareNames(a, b string) int {
	if c := strings.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}
