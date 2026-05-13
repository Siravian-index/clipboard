package ui

type UI interface {
	Show(items []string) (selected string, err error)
}
