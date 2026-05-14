package watcher

import "github.com/david-pena/clipboard/history"

type Watcher interface {
	Start(onChange func(entry history.ClipboardEntry)) error
	Stop() error
	Reset()
}
