package main

import (
	"fmt"
	"os"

	"github.com/david-pena/clipboard/client"
	"github.com/david-pena/clipboard/daemon"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "daemon":
		daemon.NewServer().Run()
	case "show":
		client.Run()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  clipboard-manager daemon   start the background daemon")
	fmt.Fprintln(os.Stderr, "  clipboard-manager show     show clipboard history picker")
}
