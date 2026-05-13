package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.design/x/clipboard"

	"github.com/david-pena/clipboard/history"
	"github.com/david-pena/clipboard/hotkey"
	"github.com/david-pena/clipboard/ui"
	"github.com/david-pena/clipboard/watcher"
)

func main() {
	hist := history.NewMemoryHistory(50)
	watch := watcher.NewPollingWatcher(500 * time.Millisecond)
	fyneUI := ui.NewFyneUI()

	hkListener, err := hotkey.NewXGBListener()
	if err != nil {
		log.Fatalf("error starting hotkey listener: %v", err)
	}

	if err := watch.Start(func(content string) {
		hist.Add(content)
		log.Printf("captured: %.40s", content)
	}); err != nil {
		log.Fatalf("error starting watcher: %v", err)
	}
	defer watch.Stop()

	err = hkListener.Register("ctrl+shift+v", func() {
		items := hist.List()
		selected, err := fyneUI.Show(items)
		if err != nil {
			return
		}
		clipboard.Write(clipboard.FmtText, []byte(selected))
		log.Printf("pasted: %.40s", selected)
	})
	if err != nil {
		log.Fatalf("error registering hotkey: %v", err)
	}

	if err := hkListener.Listen(); err != nil {
		log.Fatalf("error starting listener: %v", err)
	}
	defer hkListener.Stop()

	log.Println("clipboard manager running — Ctrl+Shift+V to open history")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("shutting down...")
}
