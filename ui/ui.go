package ui

type UI interface {
	Show(items []string, updates <-chan string) (selected string, err error)
}
