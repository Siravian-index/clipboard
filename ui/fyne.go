package ui

import (
	"errors"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type FyneUI struct{}

func NewFyneUI() *FyneUI {
	return &FyneUI{}
}

func (f *FyneUI) Show(items []string, updates <-chan string) (string, error) {
	if len(items) == 0 && updates == nil {
		return "", errors.New("no history entries")
	}

	a := app.New()
	w := a.NewWindow("Clipboard History")
	w.Resize(fyne.NewSize(500, 400))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	data := binding.NewStringList()
	_ = data.Set(items)

	selected := ""

	list := widget.NewListWithData(
		data,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(item binding.DataItem, o fyne.CanvasObject) {
			s, _ := item.(binding.String).Get()
			text := strings.ReplaceAll(s, "\n", " ")
			text = strings.Join(strings.Fields(text), " ")
			if len([]rune(text)) > 80 {
				text = string([]rune(text)[:80]) + "…"
			}
			o.(*widget.Label).SetText(text)
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		all, _ := data.Get()
		if id < len(all) {
			selected = all[id]
			w.Close()
		}
	}

	// Listen for new items from the daemon and prepend them to the list.
	if updates != nil {
		go func() {
			for item := range updates {
				current, _ := data.Get()
				_ = data.Set(append([]string{item}, current...))
			}
		}()
	}

	w.SetContent(container.NewBorder(
		widget.NewLabel("Select an entry to paste:"),
		nil, nil, nil,
		list,
	))

	w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyEscape {
			w.Close()
		}
	})

	w.ShowAndRun()

	if selected == "" {
		return "", errors.New("no entry selected")
	}
	return selected, nil
}
