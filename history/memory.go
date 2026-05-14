package history

import (
	"strings"
	"sync"
)

type MemoryHistory struct {
	mu      sync.RWMutex
	entries []ClipboardEntry
	maxSize int
}

func NewMemoryHistory(maxSize int) *MemoryHistory {
	return &MemoryHistory{maxSize: maxSize}
}

func (h *MemoryHistory) Add(entry ClipboardEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if entry.Type == EntryTypeText {
		entry.Content = strings.TrimSpace(entry.Content)
		if entry.Content == "" {
			return
		}
	}

	// Remove existing occurrence anywhere in the list before prepending.
	filtered := h.entries[:0]
	for _, e := range h.entries {
		if !(e.Type == entry.Type && e.Content == entry.Content) {
			filtered = append(filtered, e)
		}
	}

	// Assign a simple incrementing ID based on current max.
	var nextID int64 = 1
	for _, e := range filtered {
		if e.ID >= nextID {
			nextID = e.ID + 1
		}
	}
	entry.ID = nextID

	h.entries = append([]ClipboardEntry{entry}, filtered...)
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[:h.maxSize]
	}
}

func (h *MemoryHistory) List() []ClipboardEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]ClipboardEntry, len(h.entries))
	copy(result, h.entries)
	return result
}

func (h *MemoryHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = nil
}
