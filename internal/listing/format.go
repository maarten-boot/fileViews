package listing

import (
	"fmt"
	"io/fs"
	"strconv"
)

// Column identifies one column of the ls -lF style listing (FR-022).
type Column int

const (
	ColPerms Column = iota
	ColLinks
	ColOwner
	ColGroup
	ColSize
	ColModified
	ColName
	numColumns
)

// DefaultColumns is the ls -l column order.
var DefaultColumns = []Column{ColPerms, ColLinks, ColOwner, ColGroup, ColSize, ColModified, ColName}

var columnNames = [numColumns]string{"Permissions", "Links", "Owner", "Group", "Size", "Modified", "Name"}

func (c Column) String() string {
	if c < 0 || c >= numColumns {
		return "Column(" + strconv.Itoa(int(c)) + ")"
	}
	return columnNames[c]
}

// ParseColumn returns the column with the given name, as produced by String.
func ParseColumn(name string) (Column, bool) {
	for i, n := range columnNames {
		if n == name {
			return Column(i), true
		}
	}
	return 0, false
}

// ColumnLayout is the user's column order and visibility (FR-024).
type ColumnLayout struct {
	Order  []Column
	Hidden map[Column]bool
}

// Visible returns the columns to display, in order. Unknown and duplicate
// columns are dropped, columns missing from Order are appended, and the
// Name column is always included.
func (l ColumnLayout) Visible() []Column {
	var seen [numColumns]bool
	cols := make([]Column, 0, numColumns)
	add := func(c Column) {
		if c < 0 || c >= numColumns || seen[c] {
			return
		}
		seen[c] = true
		if c == ColName || !l.Hidden[c] {
			cols = append(cols, c)
		}
	}
	for _, c := range l.Order {
		add(c)
	}
	for _, c := range DefaultColumns {
		add(c)
	}
	return cols
}

// DefaultTimeLayout is the format of the Modified column.
const DefaultTimeLayout = "2006-01-02 15:04"

// FormatOptions control how fields are rendered.
type FormatOptions struct {
	ExactSizes bool   // FR-026: exact bytes instead of human-readable
	TimeLayout string // empty means DefaultTimeLayout
}

// FormatRow returns the text of each column in cols for e.
func FormatRow(e Entry, cols []Column, opts FormatOptions) []string {
	row := make([]string, len(cols))
	for i, c := range cols {
		row[i] = FormatField(e, c, opts)
	}
	return row
}

// FormatField returns the text of column c for e. Fields that could not be
// read are shown as "?" (FR-029).
func FormatField(e Entry, c Column, opts FormatOptions) string {
	if c == ColName {
		return DisplayName(e)
	}
	if e.Mode == 0 && e.Err != nil {
		return "?"
	}
	switch c {
	case ColPerms:
		return PermString(e.Mode)
	case ColLinks:
		return strconv.FormatUint(e.Links, 10)
	case ColOwner:
		return e.Owner
	case ColGroup:
		return e.Group
	case ColSize:
		if opts.ExactSizes {
			return strconv.FormatInt(e.Size, 10)
		}
		return HumanSize(e.Size)
	case ColModified:
		layout := opts.TimeLayout
		if layout == "" {
			layout = DefaultTimeLayout
		}
		return e.ModTime.Format(layout)
	}
	return ""
}

// DisplayName returns the name with its ls -F type marker (FR-023).
func DisplayName(e Entry) string {
	switch e.Kind {
	case KindDir:
		return e.Name + "/"
	case KindSymlink:
		if e.LinkTarget != "" {
			return e.Name + "@ -> " + e.LinkTarget
		}
		return e.Name + "@"
	case KindFIFO:
		return e.Name + "|"
	case KindSocket:
		return e.Name + "="
	case KindFile:
		if e.Mode&0o111 != 0 {
			return e.Name + "*"
		}
	}
	return e.Name
}

// PermString renders m like ls -l, e.g. "drwxr-xr-x", including setuid,
// setgid and sticky bits.
func PermString(m fs.FileMode) string {
	b := []byte("----------")
	switch {
	case m&fs.ModeDir != 0:
		b[0] = 'd'
	case m&fs.ModeSymlink != 0:
		b[0] = 'l'
	case m&fs.ModeNamedPipe != 0:
		b[0] = 'p'
	case m&fs.ModeSocket != 0:
		b[0] = 's'
	case m&fs.ModeCharDevice != 0:
		b[0] = 'c'
	case m&fs.ModeDevice != 0:
		b[0] = 'b'
	}
	const rwx = "rwxrwxrwx"
	for i := range 9 {
		if m&(1<<uint(8-i)) != 0 {
			b[i+1] = rwx[i]
		}
	}
	special := func(pos int, set bool, lower, upper byte) {
		if !set {
			return
		}
		if b[pos] == 'x' {
			b[pos] = lower
		} else {
			b[pos] = upper
		}
	}
	special(3, m&fs.ModeSetuid != 0, 's', 'S')
	special(6, m&fs.ModeSetgid != 0, 's', 'S')
	special(9, m&fs.ModeSticky != 0, 't', 'T')
	return string(b)
}

// HumanSize renders n like ls -h: bytes below 1024, otherwise one decimal
// below 10 and whole numbers above, with a K/M/G/T/P/E suffix.
func HumanSize(n int64) string {
	if n < 1024 {
		return strconv.FormatInt(n, 10)
	}
	const units = "KMGTPE"
	v := float64(n)
	i := -1
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if v < 9.95 {
		return fmt.Sprintf("%.1f%c", v, units[i])
	}
	return fmt.Sprintf("%.0f%c", v, units[i])
}
