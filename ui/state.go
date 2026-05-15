package ui

import (
	"sync"

	"github.com/david-pena/clipboard/history"
)

// clipboardState holds the mutable list state shared between the Fyne main
// thread and the background message-loop goroutine. All methods are safe for
// concurrent use.
type clipboardState struct {
	mu           sync.Mutex
	current      []history.ClipboardEntry
	filtered     []history.ClipboardEntry
	activeQuery  string
	totalMatches int
	totalCount   int
	maxEntries   int
}

func newClipboardState(items []history.ClipboardEntry, totalCount, maxEntries int) *clipboardState {
	s := &clipboardState{
		current:    make([]history.ClipboardEntry, len(items)),
		totalCount: totalCount,
		maxEntries: maxEntries,
	}
	copy(s.current, items)
	s.filtered = s.current
	return s
}

// AddEntry deduplicates and prepends entry. Returns the active query if one is
// set (caller should call sendSearch), or "" if filtered was updated directly.
func (s *clipboardState) AddEntry(entry history.ClipboardEntry) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	deduped := s.current[:0]
	for _, e := range s.current {
		if !(e.Type == entry.Type && e.Content == entry.Content) {
			deduped = append(deduped, e)
		}
	}
	s.current = append([]history.ClipboardEntry{entry}, deduped...)
	if s.maxEntries > 0 && len(s.current) > s.maxEntries {
		s.current = s.current[:s.maxEntries]
	}
	if s.activeQuery == "" {
		s.filtered = s.current
	}
	return s.activeQuery
}

// Refresh replaces current entries. Returns the active query if one is set.
func (s *clipboardState) Refresh(items []history.ClipboardEntry) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.current = items
	if s.activeQuery == "" {
		s.filtered = s.current
	}
	return s.activeQuery
}

// ApplySearch updates the visible filtered list from a daemon search response.
func (s *clipboardState) ApplySearch(sr SearchResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filtered = sr.Items
	s.totalMatches = sr.TotalMatches
}

// SetCount updates the total DB entry count.
func (s *clipboardState) SetCount(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalCount = n
}

// SetQuery sets the active search query. When q is empty, filtered is reset to
// current and totalMatches is cleared.
func (s *clipboardState) SetQuery(q string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeQuery = q
	if q == "" {
		s.filtered = s.current
		s.totalMatches = 0
	}
}

// Clear resets all list state (called when history is cleared by the user).
func (s *clipboardState) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = nil
	s.filtered = nil
	s.activeQuery = ""
	s.totalMatches = 0
	s.totalCount = 0
}

// FilteredCount returns the number of entries currently shown in the list.
func (s *clipboardState) FilteredCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.filtered)
}

// EntryAt returns the entry at index i in the filtered list.
// Returns false if i is out of range.
func (s *clipboardState) EntryAt(i int) (history.ClipboardEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i >= len(s.filtered) {
		return history.ClipboardEntry{}, false
	}
	return s.filtered[i], true
}

// ActiveQuery returns the current search query string.
func (s *clipboardState) ActiveQuery() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeQuery
}

// TotalCount returns the known total number of entries in the DB.
func (s *clipboardState) TotalCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.totalCount
}

// IsEmpty reports whether the full history and filtered view are empty.
func (s *clipboardState) IsEmpty() (totalEmpty, filteredEmpty bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.current) == 0, len(s.filtered) == 0
}

// ExtraMatches returns the active query and the count of search matches not
// shown in the current filtered list (used to display the "more results" hint).
func (s *clipboardState) ExtraMatches() (query string, extra int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeQuery, s.totalMatches - len(s.filtered)
}

// Current returns a snapshot copy of the full (unfiltered) entry list.
func (s *clipboardState) Current() []history.ClipboardEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := make([]history.ClipboardEntry, len(s.current))
	copy(snap, s.current)
	return snap
}
