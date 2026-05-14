package history

import (
	"testing"
)

func newTestSQLite(t *testing.T, maxSize int) *SQLiteHistory {
	t.Helper()
	h, err := NewSQLiteHistory(":memory:", t.TempDir(), maxSize)
	if err != nil {
		t.Fatalf("failed to create SQLiteHistory: %v", err)
	}
	t.Cleanup(func() { h.Close() })
	return h
}

func textEntry(content string) ClipboardEntry {
	return ClipboardEntry{Type: EntryTypeText, Content: content}
}

func TestSQLiteHistory_AddAndList(t *testing.T) {
	h := newTestSQLite(t, 10)

	h.Add(textEntry("first"))
	h.Add(textEntry("second"))
	h.Add(textEntry("third"))

	items := h.List()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].Content != "third" {
		t.Errorf("expected 'third' at index 0, got %q", items[0].Content)
	}
	if items[2].Content != "first" {
		t.Errorf("expected 'first' at index 2, got %q", items[2].Content)
	}
}

func TestSQLiteHistory_IgnoresEmptyAndWhitespace(t *testing.T) {
	h := newTestSQLite(t, 10)

	h.Add(textEntry(""))
	h.Add(textEntry("   "))
	h.Add(textEntry("\t\n"))

	if len(h.List()) != 0 {
		t.Errorf("expected empty history, got %d items", len(h.List()))
	}
}

func TestSQLiteHistory_DeduplicatesExisting(t *testing.T) {
	h := newTestSQLite(t, 10)

	h.Add(textEntry("a"))
	h.Add(textEntry("b"))
	h.Add(textEntry("c"))
	h.Add(textEntry("a")) // Should move to top, not duplicate.

	items := h.List()
	if len(items) != 3 {
		t.Fatalf("expected 3 items after dedup, got %d", len(items))
	}
	if items[0].Content != "a" {
		t.Errorf("expected 'a' at top after re-add, got %q", items[0].Content)
	}
}

func TestSQLiteHistory_RespectsMaxSize(t *testing.T) {
	h := newTestSQLite(t, 3)

	h.Add(textEntry("a"))
	h.Add(textEntry("b"))
	h.Add(textEntry("c"))
	h.Add(textEntry("d")) // Should evict "a".

	items := h.List()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	for _, item := range items {
		if item.Content == "a" {
			t.Error("expected 'a' to be evicted but it's still present")
		}
	}
}

func TestSQLiteHistory_Clear(t *testing.T) {
	h := newTestSQLite(t, 10)

	h.Add(textEntry("a"))
	h.Add(textEntry("b"))
	h.Clear()

	if len(h.List()) != 0 {
		t.Errorf("expected empty history after Clear, got %d items", len(h.List()))
	}
}

func TestSQLiteHistory_SetMaxSize(t *testing.T) {
	h := newTestSQLite(t, 10)

	h.Add(textEntry("a"))
	h.Add(textEntry("b"))
	h.Add(textEntry("c"))

	h.SetMaxSize(2)
	h.Add(textEntry("d")) // Should now evict down to 2 entries.

	items := h.List()
	if len(items) != 2 {
		t.Fatalf("expected 2 items after SetMaxSize(2), got %d", len(items))
	}
	if items[0].Content != "d" {
		t.Errorf("expected 'd' at top, got %q", items[0].Content)
	}
}

func TestSQLiteHistory_ConsecutiveDuplicateIgnored(t *testing.T) {
	h := newTestSQLite(t, 10)

	h.Add(textEntry("same"))
	h.Add(textEntry("same"))

	if len(h.List()) != 1 {
		t.Errorf("expected 1 item for consecutive duplicate, got %d", len(h.List()))
	}
}

func TestSQLiteHistory_ImageEntry(t *testing.T) {
	h := newTestSQLite(t, 10)

	h.Add(ClipboardEntry{Type: EntryTypeImage, Content: "/tmp/test.png"})

	items := h.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Type != EntryTypeImage {
		t.Errorf("expected image type, got %q", items[0].Type)
	}
	if items[0].Content != "/tmp/test.png" {
		t.Errorf("expected '/tmp/test.png', got %q", items[0].Content)
	}
}
