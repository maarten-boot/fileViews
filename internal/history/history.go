package history

// History is a browser-style back/forward list of directories (FR-012).
// The zero value is empty; use Visit to set the first directory.
type History struct {
	entries []string
	pos     int // index of the current entry; meaningless when entries is empty
}

// Current returns the current directory, or "" if nothing was visited yet.
func (h *History) Current() string {
	if len(h.entries) == 0 {
		return ""
	}
	return h.entries[h.pos]
}

// Visit makes dir the current entry and drops all forward entries.
// Visiting the current directory again is a no-op.
func (h *History) Visit(dir string) {
	if len(h.entries) > 0 && h.entries[h.pos] == dir {
		return
	}
	if len(h.entries) > 0 {
		h.entries = h.entries[:h.pos+1]
	}
	h.entries = append(h.entries, dir)
	h.pos = len(h.entries) - 1
}

// CanBack reports whether there is an entry before the current one.
func (h *History) CanBack() bool { return len(h.entries) > 0 && h.pos > 0 }

// CanForward reports whether there is an entry after the current one.
func (h *History) CanForward() bool { return h.pos < len(h.entries)-1 }

// PeekBack returns the previous entry without moving.
func (h *History) PeekBack() (string, bool) {
	if !h.CanBack() {
		return "", false
	}
	return h.entries[h.pos-1], true
}

// PeekForward returns the next entry without moving.
func (h *History) PeekForward() (string, bool) {
	if !h.CanForward() {
		return "", false
	}
	return h.entries[h.pos+1], true
}

// Back moves to the previous entry and returns it.
func (h *History) Back() (string, bool) {
	if !h.CanBack() {
		return "", false
	}
	h.pos--
	return h.entries[h.pos], true
}

// Forward moves to the next entry and returns it.
func (h *History) Forward() (string, bool) {
	if !h.CanForward() {
		return "", false
	}
	h.pos++
	return h.entries[h.pos], true
}
