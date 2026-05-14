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
	h := NewMemoryHistory(3)

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
