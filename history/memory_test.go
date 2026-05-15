package history

import (
	"testing"
)

func TestMemoryHistory_AddAndList(t *testing.T) {
	h := NewMemoryHistory(10)

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

func TestMemoryHistory_IgnoresEmptyAndWhitespace(t *testing.T) {
	h := NewMemoryHistory(10)

	h.Add(textEntry(""))
	h.Add(textEntry("   "))
	h.Add(textEntry("\t\n"))

	if len(h.List()) != 0 {
		t.Errorf("expected empty history, got %d items", len(h.List()))
	}
}

func TestMemoryHistory_DeduplicatesExisting(t *testing.T) {
	h := NewMemoryHistory(10)

	h.Add(textEntry("a"))
	h.Add(textEntry("b"))
	h.Add(textEntry("c"))
	h.Add(textEntry("a")) // "a" already exists — should move to top, not duplicate.

	items := h.List()
	if len(items) != 3 {
		t.Fatalf("expected 3 items after dedup, got %d", len(items))
	}
	if items[0].Content != "a" {
		t.Errorf("expected 'a' at top after re-add, got %q", items[0].Content)
	}
}

func TestMemoryHistory_RespectsMaxSize(t *testing.T) {
	// MemoryHistory no longer trims entries — maxSize is a visual limit only.
	h := NewMemoryHistory(3)

	h.Add(textEntry("a"))
	h.Add(textEntry("b"))
	h.Add(textEntry("c"))
	h.Add(textEntry("d"))

	items := h.List()
	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(items))
	}
}

func TestMemoryHistory_Clear(t *testing.T) {
	h := NewMemoryHistory(10)

	h.Add(textEntry("a"))
	h.Add(textEntry("b"))
	h.Clear()

	if len(h.List()) != 0 {
		t.Errorf("expected empty history after Clear, got %d items", len(h.List()))
	}
}

func TestMemoryHistory_ConsecutiveDuplicateIgnored(t *testing.T) {
	h := NewMemoryHistory(10)

	h.Add(textEntry("same"))
	h.Add(textEntry("same"))

	if len(h.List()) != 1 {
		t.Errorf("expected 1 item for consecutive duplicate, got %d", len(h.List()))
	}
}

func TestMemoryHistory_TrimPreservesWhitespace(t *testing.T) {
	h := NewMemoryHistory(10)

	h.Add(textEntry("  hello  "))

	items := h.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Content != "hello" {
		t.Errorf("expected trimmed value 'hello', got %q", items[0].Content)
	}
}

func TestMemoryHistory_Count(t *testing.T) {
	h := NewMemoryHistory(10)

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

func TestMemoryHistory_Search(t *testing.T) {
	h := NewMemoryHistory(10)
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

	t.Run("case insensitive", func(t *testing.T) {
		r := h.Search("HELLO", 10)
		if r.TotalMatches != 2 {
			t.Errorf("expected 2 matches case-insensitive, got %d", r.TotalMatches)
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
