package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/aholstenson/logseq-go"
)

func main() {
	// Read the directory to use for the graph.
	var directory string
	flag.StringVar(&directory, "directory", "", "Directory to open")

	// Parse the command line flags.
	flag.Parse()

	if directory == "" {
		println("--directory is required")
		return
	}

	ctx := context.Background()
	graph, err := logseq.Open(ctx, directory)
	if err != nil {
		println("Failed to open graph:", err.Error())
		return
	}
	defer graph.Close()

	println("Will print changes")

	watcher := graph.Watch()
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-watcher.Events():
			switch event := e.(type) {
			case *logseq.PageUpdated:
				fmt.Printf("Page updated: %s\n", event.Page.Title())
			case *logseq.PageDeleted:
				fmt.Printf("Page deleted: %s\n", event.Title)
			}
		}
	}
}
