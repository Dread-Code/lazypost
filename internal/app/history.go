package app

import (
	"time"

	"github.com/Dread-Code/lazypost/internal/collection"
	"github.com/Dread-Code/lazypost/internal/httpclient"
)

// HistoryEntry is one send kept for inspection or resend: the full request
// snapshot, the result summary, and the response itself (so selecting an
// entry restores the response pane). Errors carry Err instead of Res.
// The ring is memory-bounded by its cap ([[Design - request history]]).
type HistoryEntry struct {
	Req     collection.Request
	Path    string // source path, empty for unsaved requests
	Summary string // res.Summary() on success, err.Error() on failure
	At      time.Time
	Res     *httpclient.Response // non-nil on success
	Err     error                // non-nil on failure
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
	entries := make([]HistoryEntry, len(h.entries))
	copy(entries, h.entries)
	return entries
}
