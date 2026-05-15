package history

import (
	"os"
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

func TestSQLiteHistory_Count(t *testing.T) {
	h := newTestSQLite(t, 10)

	if h.Count() != 0 {
		t.Errorf("expected 0, got %d", h.Count())
	}

	h.Add(textEntry("a"))
	h.Add(textEntry("b"))

	if h.Count() != 2 {
		t.Errorf("expected 2, got %d", h.Count())
	}

	h.Clear()
	if h.Count() != 0 {
		t.Errorf("expected 0 after clear, got %d", h.Count())
	}
}

func TestSQLiteHistory_MaxSize(t *testing.T) {
	h := newTestSQLite(t, 42)

	if h.MaxSize() != 42 {
		t.Errorf("expected 42, got %d", h.MaxSize())
	}

	h.SetMaxSize(7)
	if h.MaxSize() != 7 {
		t.Errorf("expected 7 after SetMaxSize, got %d", h.MaxSize())
	}
}

func TestSQLiteHistory_ImageDir(t *testing.T) {
	dir := t.TempDir()
	h, err := NewSQLiteHistory(":memory:", dir, 10)
	if err != nil {
		t.Fatalf("failed to create SQLiteHistory: %v", err)
	}
	defer h.Close()

	if h.ImageDir() != dir {
		t.Errorf("expected %q, got %q", dir, h.ImageDir())
	}
}

func TestSQLiteHistory_Search(t *testing.T) {
	h := newTestSQLite(t, 100)
	h.Add(textEntry("hello world"))
	h.Add(textEntry("foo bar"))
	h.Add(textEntry("hello go"))

	t.Run("matches substring", func(t *testing.T) {
		r := h.Search("hello", 10)
		if r.TotalMatches != 2 {
			t.Errorf("expected 2 total matches, got %d", r.TotalMatches)
		}
		if len(r.Entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(r.Entries))
		}
	})

	t.Run("limit truncates results", func(t *testing.T) {
		r := h.Search("hello", 1)
		if len(r.Entries) != 1 {
			t.Errorf("expected 1 entry after limit, got %d", len(r.Entries))
		}
		if r.TotalMatches != 2 {
			t.Errorf("expected TotalMatches=2 even when limited, got %d", r.TotalMatches)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		r := h.Search("zzz", 10)
		if r.TotalMatches != 0 || len(r.Entries) != 0 {
			t.Errorf("expected empty result, got %+v", r)
		}
	})
}

func TestEnsureImageDir(t *testing.T) {
	base := t.TempDir()
	dir, err := EnsureImageDir(base)
	if err != nil {
		t.Fatalf("EnsureImageDir failed: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("image dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory")
	}

	// Idempotent — calling again should not error.
	if _, err := EnsureImageDir(base); err != nil {
		t.Errorf("second call to EnsureImageDir failed: %v", err)
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
