package history

import (
	"strings"
	"sync"
)

type MemoryHistory struct {
	mu      sync.RWMutex
	entries []string
	maxSize int
}

func NewMemoryHistory(maxSize int) *MemoryHistory {
	return &MemoryHistory{maxSize: maxSize}
}

func (h *MemoryHistory) Add(entry string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry = strings.TrimSpace(entry)
	if entry == "" {
		return
	}

	// Remove existing occurrence anywhere in the list before prepending.
	filtered := h.entries[:0]
	for _, e := range h.entries {
		if e != entry {
			filtered = append(filtered, e)
		}
	}

	h.entries = append([]string{entry}, filtered...)
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[:h.maxSize]
	}
}

func (h *MemoryHistory) List() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]string, len(h.entries))
	copy(result, h.entries)
	return result
}

func (h *MemoryHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = nil
}
