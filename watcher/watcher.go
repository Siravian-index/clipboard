package watcher

type Watcher interface {
	Start(onChange func(content string)) error
	Stop() error
}
