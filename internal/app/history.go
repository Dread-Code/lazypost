package app

import (
	"time"

	"lazypost/internal/collection"
)

// HistoryEntry is one send kept for inspection or resend: the full request
// snapshot plus a one-line result summary. The response body never enters
// history — summaries only, so the ring stays memory-bounded
// ([[Design - request history]]).
type HistoryEntry struct {
	Req     collection.Request
	Summary string // res.Summary() on success, err.Error() on failure
	At      time.Time
}

// History is a bounded, newest-last ring of sends.
type History struct {
	entries []HistoryEntry
	cap     int
}

// NewHistory returns a history that keeps at most cap entries.
func NewHistory(cap int) *History {
	return &History{cap: cap}
}

// Add appends an entry, dropping the oldest once the ring is full.
func (h *History) Add(e HistoryEntry) {
	if h.cap <= 0 {
		return
	}
	if len(h.entries) == h.cap {
		h.entries = h.entries[1:]
	}
	h.entries = append(h.entries, e)
}

// List returns the entries oldest first.
func (h *History) List() []HistoryEntry {
	return h.entries
}
