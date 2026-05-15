package ui

import (
	"testing"

	"github.com/david-pena/clipboard/history"
)

func textClip(content string) history.ClipboardEntry {
	return history.ClipboardEntry{Type: history.EntryTypeText, Content: content}
}

func imageClip(path string) history.ClipboardEntry {
	return history.ClipboardEntry{Type: history.EntryTypeImage, Content: path}
}

// --- newClipboardState ---

func TestNewClipboardState_InitializesFiltered(t *testing.T) {
	items := []history.ClipboardEntry{textClip("a"), textClip("b")}
	s := newClipboardState(items, 42, 100)

	if s.FilteredCount() != 2 {
		t.Errorf("expected 2 filtered entries, got %d", s.FilteredCount())
	}
	if s.TotalCount() != 42 {
		t.Errorf("expected totalCount=42, got %d", s.TotalCount())
	}
}

func TestNewClipboardState_EmptyItems(t *testing.T) {
	s := newClipboardState(nil, 0, 100)
	if s.FilteredCount() != 0 {
		t.Errorf("expected 0 filtered, got %d", s.FilteredCount())
	}
}

// --- AddEntry ---

func TestAddEntry_PrependsEntry(t *testing.T) {
	s := newClipboardState([]history.ClipboardEntry{textClip("old")}, 1, 100)
	q := s.AddEntry(textClip("new"))

	if q != "" {
		t.Errorf("expected no active query, got %q", q)
	}
	entry, ok := s.EntryAt(0)
	if !ok || entry.Content != "new" {
		t.Errorf("expected 'new' at index 0, got %q", entry.Content)
	}
	if s.FilteredCount() != 2 {
		t.Errorf("expected 2 entries, got %d", s.FilteredCount())
	}
}

func TestAddEntry_DeduplicatesExisting(t *testing.T) {
	s := newClipboardState([]history.ClipboardEntry{textClip("a"), textClip("b")}, 2, 100)
	s.AddEntry(textClip("a"))

	if s.FilteredCount() != 2 {
		t.Errorf("expected 2 entries after dedup, got %d", s.FilteredCount())
	}
	entry, _ := s.EntryAt(0)
	if entry.Content != "a" {
		t.Errorf("expected 'a' at top after re-add, got %q", entry.Content)
	}
}

func TestAddEntry_RespectsMaxEntries(t *testing.T) {
	items := []history.ClipboardEntry{textClip("a"), textClip("b"), textClip("c")}
	s := newClipboardState(items, 3, 3)
	s.AddEntry(textClip("d"))

	if s.FilteredCount() != 3 {
		t.Errorf("expected 3 entries (maxEntries), got %d", s.FilteredCount())
	}
	entry, _ := s.EntryAt(0)
	if entry.Content != "d" {
		t.Errorf("expected 'd' at top, got %q", entry.Content)
	}
}

func TestAddEntry_ReturnsQueryWhenActive(t *testing.T) {
	s := newClipboardState(nil, 0, 100)
	s.SetQuery("hello")
	q := s.AddEntry(textClip("world"))

	if q != "hello" {
		t.Errorf("expected active query 'hello', got %q", q)
	}
	// filtered should NOT be updated when query is active
	if s.FilteredCount() != 0 {
		t.Errorf("expected filtered unchanged when query active, got %d", s.FilteredCount())
	}
}

func TestAddEntry_ZeroMaxEntriesNoTrim(t *testing.T) {
	s := newClipboardState(nil, 0, 0)
	for i := 0; i < 10; i++ {
		s.AddEntry(textClip(string(rune('a' + i))))
	}
	if s.FilteredCount() != 10 {
		t.Errorf("expected 10 entries with maxEntries=0, got %d", s.FilteredCount())
	}
}

// --- Refresh ---

func TestRefresh_ReplacesCurrent(t *testing.T) {
	s := newClipboardState([]history.ClipboardEntry{textClip("old")}, 1, 100)
	newItems := []history.ClipboardEntry{textClip("x"), textClip("y")}
	q := s.Refresh(newItems)

	if q != "" {
		t.Errorf("expected no active query, got %q", q)
	}
	if s.FilteredCount() != 2 {
		t.Errorf("expected 2 entries after refresh, got %d", s.FilteredCount())
	}
}

func TestRefresh_ReturnsQueryWhenActive(t *testing.T) {
	s := newClipboardState(nil, 0, 100)
	s.SetQuery("test")
	q := s.Refresh([]history.ClipboardEntry{textClip("item")})

	if q != "test" {
		t.Errorf("expected 'test', got %q", q)
	}
}

// --- ApplySearch ---

func TestApplySearch_UpdatesFiltered(t *testing.T) {
	s := newClipboardState([]history.ClipboardEntry{textClip("a"), textClip("b")}, 2, 100)
	s.SetQuery("a")
	s.ApplySearch(SearchResponse{
		Items:        []history.ClipboardEntry{textClip("a")},
		TotalMatches: 5,
	})

	if s.FilteredCount() != 1 {
		t.Errorf("expected 1 filtered entry, got %d", s.FilteredCount())
	}
	_, extra := s.ExtraMatches()
	if extra != 4 {
		t.Errorf("expected 4 extra matches, got %d", extra)
	}
}

// --- SetQuery ---

func TestSetQuery_EmptyResetsFiltered(t *testing.T) {
	s := newClipboardState([]history.ClipboardEntry{textClip("a"), textClip("b")}, 2, 100)
	s.SetQuery("x")
	s.ApplySearch(SearchResponse{Items: []history.ClipboardEntry{textClip("a")}, TotalMatches: 1})

	s.SetQuery("")

	if s.FilteredCount() != 2 {
		t.Errorf("expected 2 entries after clearing query, got %d", s.FilteredCount())
	}
	if s.ActiveQuery() != "" {
		t.Errorf("expected empty query, got %q", s.ActiveQuery())
	}
}

// --- SetCount / TotalCount ---

func TestSetCount(t *testing.T) {
	s := newClipboardState(nil, 0, 100)
	s.SetCount(99)
	if s.TotalCount() != 99 {
		t.Errorf("expected 99, got %d", s.TotalCount())
	}
}

// --- Clear ---

func TestClear_ResetsAllState(t *testing.T) {
	s := newClipboardState([]history.ClipboardEntry{textClip("a")}, 10, 100)
	s.SetQuery("q")
	s.ApplySearch(SearchResponse{Items: []history.ClipboardEntry{textClip("a")}, TotalMatches: 3})

	s.Clear()

	if s.FilteredCount() != 0 {
		t.Errorf("expected 0 filtered after clear, got %d", s.FilteredCount())
	}
	if s.ActiveQuery() != "" {
		t.Errorf("expected empty query after clear, got %q", s.ActiveQuery())
	}
	if s.TotalCount() != 0 {
		t.Errorf("expected 0 total count after clear, got %d", s.TotalCount())
	}
	totalEmpty, filteredEmpty := s.IsEmpty()
	if !totalEmpty || !filteredEmpty {
		t.Error("expected both empty after clear")
	}
}

// --- EntryAt ---

func TestEntryAt_OutOfRange(t *testing.T) {
	s := newClipboardState(nil, 0, 100)
	_, ok := s.EntryAt(0)
	if ok {
		t.Error("expected false for out-of-range index")
	}
}

// --- IsEmpty ---

func TestIsEmpty(t *testing.T) {
	s := newClipboardState(nil, 0, 100)
	totalEmpty, filteredEmpty := s.IsEmpty()
	if !totalEmpty || !filteredEmpty {
		t.Error("expected both empty on fresh state")
	}

	s.AddEntry(textClip("x"))
	totalEmpty, filteredEmpty = s.IsEmpty()
	if totalEmpty || filteredEmpty {
		t.Error("expected neither empty after adding entry")
	}
}

// --- ExtraMatches ---

func TestExtraMatches_NoQuery(t *testing.T) {
	s := newClipboardState(nil, 0, 100)
	q, extra := s.ExtraMatches()
	if q != "" || extra != 0 {
		t.Errorf("expected empty query and 0 extra, got %q %d", q, extra)
	}
}

// --- Current ---

func TestCurrent_ReturnsCopy(t *testing.T) {
	items := []history.ClipboardEntry{textClip("a"), textClip("b")}
	s := newClipboardState(items, 2, 100)

	snap := s.Current()
	snap[0].Content = "mutated"

	// Original state should be unaffected.
	entry, _ := s.EntryAt(0)
	if entry.Content == "mutated" {
		t.Error("Current() should return a copy, not a reference")
	}
}
