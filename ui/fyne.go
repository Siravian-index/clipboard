package ui

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type FyneUI struct{}

func NewFyneUI() *FyneUI {
	return &FyneUI{}
}

func (f *FyneUI) Show(items []string) (string, error) {
	if len(items) == 0 {
		return "", errors.New("no history entries")
	}

	a := app.New()
	w := a.NewWindow("Clipboard History")
	w.Resize(fyne.NewSize(500, 400))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	selected := ""

	list := widget.NewList(
		func() int { return len(items) },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			text := items[i]
			if len(text) > 80 {
				text = text[:80] + "…"
			}
			o.(*widget.Label).SetText(text)
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		selected = items[id]
		w.Close()
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
