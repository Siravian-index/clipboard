package history

import (
	"testing"
)

func newTestSQLite(t *testing.T, maxSize int) *SQLiteHistory {
	t.Helper()
	h, err := NewSQLiteHistory(":memory:", maxSize)
	if err != nil {
		t.Fatalf("failed to create SQLiteHistory: %v", err)
	}
	t.Cleanup(func() { h.Close() })
	return h
}

func TestSQLiteHistory_AddAndList(t *testing.T) {
	h := newTestSQLite(t, 10)

	h.Add("first")
	h.Add("second")
	h.Add("third")

	items := h.List()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0] != "third" {
		t.Errorf("expected 'third' at index 0, got %q", items[0])
	}
	if items[2] != "first" {
		t.Errorf("expected 'first' at index 2, got %q", items[2])
	}
}

func TestSQLiteHistory_IgnoresEmptyAndWhitespace(t *testing.T) {
	h := newTestSQLite(t, 10)

	h.Add("")
	h.Add("   ")
	h.Add("\t\n")

	if len(h.List()) != 0 {
		t.Errorf("expected empty history, got %d items", len(h.List()))
	}
}

func TestSQLiteHistory_DeduplicatesExisting(t *testing.T) {
	h := newTestSQLite(t, 10)

	h.Add("a")
	h.Add("b")
	h.Add("c")
	h.Add("a") // Should move to top, not duplicate.

	items := h.List()
	if len(items) != 3 {
		t.Fatalf("expected 3 items after dedup, got %d", len(items))
	}
	if items[0] != "a" {
		t.Errorf("expected 'a' at top after re-add, got %q", items[0])
	}
}

func TestSQLiteHistory_RespectsMaxSize(t *testing.T) {
	h := newTestSQLite(t, 3)

	h.Add("a")
	h.Add("b")
	h.Add("c")
	h.Add("d") // Should evict "a".

	items := h.List()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	for _, item := range items {
		if item == "a" {
			t.Error("expected 'a' to be evicted but it's still present")
		}
	}
}

func TestSQLiteHistory_Clear(t *testing.T) {
	h := newTestSQLite(t, 10)

	h.Add("a")
	h.Add("b")
	h.Clear()

	if len(h.List()) != 0 {
		t.Errorf("expected empty history after Clear, got %d items", len(h.List()))
	}
}

func TestSQLiteHistory_SetMaxSize(t *testing.T) {
	h := newTestSQLite(t, 10)

	h.Add("a")
	h.Add("b")
	h.Add("c")

	h.SetMaxSize(2)
	h.Add("d") // Should now evict down to 2 entries.

	items := h.List()
	if len(items) != 2 {
		t.Fatalf("expected 2 items after SetMaxSize(2), got %d", len(items))
	}
	if items[0] != "d" {
		t.Errorf("expected 'd' at top, got %q", items[0])
	}
}

func TestSQLiteHistory_ConsecutiveDuplicateIgnored(t *testing.T) {
	h := newTestSQLite(t, 10)

	h.Add("same")
	h.Add("same")

	if len(h.List()) != 1 {
		t.Errorf("expected 1 item for consecutive duplicate, got %d", len(h.List()))
	}
}
