package listing

// Stats are the directory statistics shown in the status bar (FR-050).
type Stats struct {
	Dirs      int   // shown directories (including symlinks to directories)
	Files     int   // shown non-directory entries
	Hidden    int   // all hidden entries, shown or not
	TotalSize int64 // sum of sizes of shown non-directory entries
}

// ComputeStats computes Stats over all entries of a directory, honouring
// showHidden for everything except the Hidden count.
func ComputeStats(all []Entry, showHidden bool) Stats {
	var s Stats
	for _, e := range all {
		if e.Hidden() {
			s.Hidden++
			if !showHidden {
				continue
			}
		}
		if e.IsDirLike() {
			s.Dirs++
		} else {
			s.Files++
			s.TotalSize += e.Size
		}
	}
	return s
}

// TotalSize returns the summed size of the non-directory entries, e.g. a selection.
func TotalSize(entries []Entry) int64 {
	var n int64
	for _, e := range entries {
		if !e.IsDirLike() {
			n += e.Size
		}
	}
	return n
}
