package history

type History interface {
	Add(entry string)
	List() []string
	Clear()
}
