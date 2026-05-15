package ui

import (
	"testing"
	"time"

	"github.com/david-pena/clipboard/history"
)

// helpers to build channels with pre-loaded values and then close them.
func updatesOf(entries ...history.ClipboardEntry) chan history.ClipboardEntry {
	ch := make(chan history.ClipboardEntry, len(entries))
	for _, e := range entries {
		ch <- e
	}
	return ch
}

func refreshesOf(batches ...[]history.ClipboardEntry) chan []history.ClipboardEntry {
	ch := make(chan []history.ClipboardEntry, len(batches))
	for _, b := range batches {
		ch <- b
	}
	return ch
}

func searchesOf(results ...SearchResponse) chan SearchResponse {
	ch := make(chan SearchResponse, len(results))
	for _, r := range results {
		ch <- r
	}
	return ch
}

func countsOf(ns ...int) chan int {
	ch := make(chan int, len(ns))
	for _, n := range ns {
		ch <- n
	}
	return ch
}

// runLoop runs runMessageLoop in a goroutine and returns a done channel that
// closes when the loop exits.
func runLoop(
	state *clipboardState,
	updates chan history.ClipboardEntry,
	refreshes chan []history.ClipboardEntry,
	searches chan SearchResponse,
	counts chan int,
	focusReqs chan struct{},
	sendSearch func(string),
	cbs loopCallbacks,
) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		runMessageLoop(state, updates, refreshes, searches, counts, focusReqs, sendSearch, cbs)
		close(done)
	}()
	return done
}

func waitDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for loop to exit")
	}
}

// --- updates channel ---

func TestLoop_UpdateAddsEntry(t *testing.T) {
	s := newClipboardState(nil, 0, 100)
	listRefreshed := make(chan struct{}, 1)
	cbs := loopCallbacks{
		RefreshList:  func() { listRefreshed <- struct{}{} },
		RefreshEmpty: func() {},
		RefreshMore:  func() {},
		UpdateCount:  func(int) {},
		RequestFocus: func() {},
	}

	updates := updatesOf(textClip("hello"))
	close(updates)

	done := runLoop(s, updates, nil, make(chan SearchResponse), make(chan int), make(chan struct{}), func(string) {}, cbs)
	waitDone(t, done)

	if s.FilteredCount() != 1 {
		t.Errorf("expected 1 entry after update, got %d", s.FilteredCount())
	}
	select {
	case <-listRefreshed:
	default:
		t.Error("expected RefreshList to be called")
	}
}

func TestLoop_UpdateWithActiveQueryCallsSendSearch(t *testing.T) {
	s := newClipboardState(nil, 0, 100)
	s.SetQuery("foo")

	searched := make(chan string, 1)
	cbs := loopCallbacks{
		RefreshList:  func() {},
		RefreshEmpty: func() {},
		RefreshMore:  func() {},
		UpdateCount:  func(int) {},
		RequestFocus: func() {},
	}

	updates := updatesOf(textClip("foobar"))
	close(updates)

	done := runLoop(s, updates, nil, make(chan SearchResponse), make(chan int), make(chan struct{}),
		func(q string) { searched <- q }, cbs)
	waitDone(t, done)

	select {
	case q := <-searched:
		if q != "foo" {
			t.Errorf("expected sendSearch('foo'), got %q", q)
		}
	default:
		t.Error("expected sendSearch to be called")
	}
}

// --- refreshes channel ---

func TestLoop_RefreshReplacesCurrent(t *testing.T) {
	s := newClipboardState([]history.ClipboardEntry{textClip("old")}, 1, 100)
	cbs := loopCallbacks{
		RefreshList:  func() {},
		RefreshEmpty: func() {},
		RefreshMore:  func() {},
		UpdateCount:  func(int) {},
		RequestFocus: func() {},
	}

	refreshes := refreshesOf([]history.ClipboardEntry{textClip("new1"), textClip("new2")})
	close(refreshes)
	updates := make(chan history.ClipboardEntry)
	close(updates)

	done := runLoop(s, updates, refreshes, make(chan SearchResponse), make(chan int), make(chan struct{}), func(string) {}, cbs)
	waitDone(t, done)

	if s.FilteredCount() != 2 {
		t.Errorf("expected 2 entries after refresh, got %d", s.FilteredCount())
	}
}

func TestLoop_NilRefreshesChannelHandled(t *testing.T) {
	s := newClipboardState(nil, 0, 100)
	cbs := loopCallbacks{
		RefreshList:  func() {},
		RefreshEmpty: func() {},
		RefreshMore:  func() {},
		UpdateCount:  func(int) {},
		RequestFocus: func() {},
	}

	updates := make(chan history.ClipboardEntry)
	close(updates)

	// refreshes=nil: loop should handle it without panic (treated as never-ready)
	// We pass a pre-closed channel to not block.
	refreshes := make(chan []history.ClipboardEntry)
	close(refreshes)

	done := runLoop(s, updates, refreshes, make(chan SearchResponse), make(chan int), make(chan struct{}), func(string) {}, cbs)
	waitDone(t, done)
}

// --- searches channel ---

func TestLoop_SearchAppliesResults(t *testing.T) {
	s := newClipboardState([]history.ClipboardEntry{textClip("a"), textClip("b")}, 2, 100)
	s.SetQuery("a")

	listRefreshed := make(chan struct{}, 1)
	cbs := loopCallbacks{
		RefreshList:  func() { listRefreshed <- struct{}{} },
		RefreshEmpty: func() {},
		RefreshMore:  func() {},
		UpdateCount:  func(int) {},
		RequestFocus: func() {},
	}

	sr := SearchResponse{Items: []history.ClipboardEntry{textClip("a")}, TotalMatches: 1}
	searches := searchesOf(sr)
	close(searches) // closing searches exits the loop after processing the result

	// Use a nil updates channel so it never fires, ensuring the search is processed first.
	done := runLoop(s, nil, nil, searches, make(chan int), make(chan struct{}), func(string) {}, cbs)
	waitDone(t, done)

	if s.FilteredCount() != 1 {
		t.Errorf("expected 1 filtered entry after search, got %d", s.FilteredCount())
	}
	select {
	case <-listRefreshed:
	default:
		t.Error("expected RefreshList called after search result")
	}
}

// --- counts channel ---

func TestLoop_CountUpdatesState(t *testing.T) {
	s := newClipboardState(nil, 0, 100)
	counted := make(chan int, 1)
	focusReqs := make(chan struct{})
	cbs := loopCallbacks{
		RefreshList:  func() {},
		RefreshEmpty: func() {},
		RefreshMore:  func() {},
		UpdateCount: func(n int) {
			counted <- n
			close(focusReqs) // exit the loop after the count is processed
		},
		RequestFocus: func() {},
	}

	counts := countsOf(77)

	// nil updates: never fires, ensuring the count is processed before exit.
	done := runLoop(s, nil, nil, make(chan SearchResponse), counts, focusReqs, func(string) {}, cbs)
	waitDone(t, done)

	if s.TotalCount() != 77 {
		t.Errorf("expected totalCount=77, got %d", s.TotalCount())
	}
	select {
	case n := <-counted:
		if n != 77 {
			t.Errorf("expected UpdateCount(77), got %d", n)
		}
	default:
		t.Error("expected UpdateCount to be called")
	}
}

// --- focusReqs channel ---

func TestLoop_FocusRequestCallsCallback(t *testing.T) {
	s := newClipboardState(nil, 0, 100)
	focused := make(chan struct{}, 1)
	focusReqs := make(chan struct{}, 1)
	cbs := loopCallbacks{
		RefreshList:  func() {},
		RefreshEmpty: func() {},
		RefreshMore:  func() {},
		UpdateCount:  func(int) {},
		RequestFocus: func() {
			focused <- struct{}{}
			close(focusReqs) // exit after the focus is processed
		},
	}

	focusReqs <- struct{}{} // one focus request

	// nil updates: never fires, ensuring focus is processed before exit.
	done := runLoop(s, nil, nil, make(chan SearchResponse), make(chan int), focusReqs, func(string) {}, cbs)
	waitDone(t, done)

	select {
	case <-focused:
	default:
		t.Error("expected RequestFocus to be called")
	}
}
