package ui

import "github.com/david-pena/clipboard/history"

// loopCallbacks are the UI-side effects triggered by the background message
// loop. Each function is called from the goroutine; callers are responsible
// for wrapping them with fyne.Do when Fyne thread-safety is required.
type loopCallbacks struct {
	RefreshList  func()
	RefreshEmpty func()
	RefreshMore  func()
	UpdateCount  func(int)
	RequestFocus func()
}

// runMessageLoop processes incoming daemon messages and updates state until
// the updates or focusReqs channel is closed.
func runMessageLoop(
	state *clipboardState,
	updates <-chan history.ClipboardEntry,
	refreshes <-chan []history.ClipboardEntry,
	searches <-chan SearchResponse,
	counts <-chan int,
	focusReqs <-chan struct{},
	sendSearch func(string),
	cbs loopCallbacks,
) {
	for {
		select {
		case entry, ok := <-updates:
			if !ok {
				return
			}
			if q := state.AddEntry(entry); q != "" {
				sendSearch(q)
			} else {
				cbs.RefreshList()
				cbs.RefreshEmpty()
				cbs.RefreshMore()
			}

		case newItems, ok := <-refreshes:
			if !ok {
				refreshes = nil
				continue
			}
			if q := state.Refresh(newItems); q != "" {
				sendSearch(q)
			} else {
				cbs.RefreshList()
				cbs.RefreshEmpty()
				cbs.RefreshMore()
			}

		case sr, ok := <-searches:
			if !ok {
				return
			}
			state.ApplySearch(sr)
			cbs.RefreshList()
			cbs.RefreshEmpty()
			cbs.RefreshMore()

		case n, ok := <-counts:
			if !ok {
				counts = nil
				continue
			}
			state.SetCount(n)
			cbs.UpdateCount(n)

		case _, ok := <-focusReqs:
			if !ok {
				return
			}
			cbs.RequestFocus()
		}
	}
}
