package history

import (
	"testing"
)

func TestMemoryHistory_AddAndList(t *testing.T) {
	h := NewMemoryHistory(10)

	h.Add("first")
	h.Add("second")
	h.Add("third")

	items := h.List()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	// Most recent first.
	if items[0] != "third" {
		t.Errorf("expected 'third' at index 0, got %q", items[0])
	}
	if items[2] != "first" {
		t.Errorf("expected 'first' at index 2, got %q", items[2])
	}
}

func TestMemoryHistory_IgnoresEmptyAndWhitespace(t *testing.T) {
	h := NewMemoryHistory(10)

	h.Add("")
	h.Add("   ")
	h.Add("\t\n")

	if len(h.List()) != 0 {
		t.Errorf("expected empty history, got %d items", len(h.List()))
	}
}

func TestMemoryHistory_DeduplicatesExisting(t *testing.T) {
	h := NewMemoryHistory(10)

	h.Add("a")
	h.Add("b")
	h.Add("c")
	h.Add("a") // "a" already exists — should move to top, not duplicate.

	items := h.List()
	if len(items) != 3 {
		t.Fatalf("expected 3 items after dedup, got %d", len(items))
	}
	if items[0] != "a" {
		t.Errorf("expected 'a' at top after re-add, got %q", items[0])
	}
}

func TestMemoryHistory_RespectsMaxSize(t *testing.T) {
	h := NewMemoryHistory(3)

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

func TestMemoryHistory_Clear(t *testing.T) {
	h := NewMemoryHistory(10)

	h.Add("a")
	h.Add("b")
	h.Clear()

	if len(h.List()) != 0 {
		t.Errorf("expected empty history after Clear, got %d items", len(h.List()))
	}
}

func TestMemoryHistory_ConsecutiveDuplicateIgnored(t *testing.T) {
	h := NewMemoryHistory(10)

	h.Add("same")
	h.Add("same")

	if len(h.List()) != 1 {
		t.Errorf("expected 1 item for consecutive duplicate, got %d", len(h.List()))
	}
}

func TestMemoryHistory_TrimPreservesWhitespace(t *testing.T) {
	h := NewMemoryHistory(10)

	h.Add("  hello  ")

	items := h.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0] != "hello" {
		t.Errorf("expected trimmed value 'hello', got %q", items[0])
	}
}
