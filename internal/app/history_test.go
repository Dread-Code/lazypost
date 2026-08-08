package app

import (
	"testing"
	"time"

	"lazypost/internal/collection"
)

func TestHistoryAddAndOrder(t *testing.T) {
	h := NewHistory(5)
	for i := 0; i < 3; i++ {
		h.Add(HistoryEntry{
			Req:     collection.Request{Name: "req", Method: "GET", URL: "https://api.test"},
			Summary: "200 OK",
			At:      time.Unix(int64(i), 0),
		})
	}
	got := h.List()
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	for i, e := range got {
		if e.At.Unix() != int64(i) {
			t.Errorf("entry %d out of order: %v", i, e.At)
		}
	}
}

func TestHistoryEvictsOldest(t *testing.T) {
	h := NewHistory(2)
	for i := 0; i < 5; i++ {
		h.Add(HistoryEntry{At: time.Unix(int64(i), 0)})
	}
	got := h.List()
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].At.Unix() != 3 || got[1].At.Unix() != 4 {
		t.Errorf("expected entries 3 and 4, got %v and %v", got[0].At, got[1].At)
	}
}

func TestHistoryZeroCap(t *testing.T) {
	h := NewHistory(0)
	h.Add(HistoryEntry{})
	if len(h.List()) != 0 {
		t.Error("zero-cap history should keep nothing")
	}
}
