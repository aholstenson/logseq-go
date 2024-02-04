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

	var indexDirectory string
	flag.StringVar(&indexDirectory, "index", "", "Directory to use for the index, leave blank for in-memory index")

	// Parse the command line flags.
	flag.Parse()

	if directory == "" {
		println("--directory is required")
		return
	}

	indexOpt := logseq.WithInMemoryIndex()
	if indexDirectory != "" {
		indexOpt = logseq.WithIndex(indexDirectory)
	}

	graph, err := logseq.Open(directory, indexOpt, logseq.WithSyncListener(func(subPath string) {
		println("Synced:", subPath)
	}))
	if err != nil {
		println("Failed to open graph:", err.Error())
		return
	}
	defer graph.Close()

	println("Ready to use query graph titles. Type 'exit' or 'quit' to exit.")

	ctx := context.Background()

	for {
		// Read the query
		var query string
		print("> ")
		_, err := fmt.Scanln(&query)
		if err != nil {
			println("Failed to read query:", err.Error())
			return
		}

		if query == "exit" || query == "quit" {
			break
		}

		// Perform the query
		pages, err := graph.List(ctx, logseq.TitleMatches(query))
		if err != nil {
			println("Failed to list pages:", err.Error())
			return
		}

		for {
			page, err := pages.Next()
			if err != nil {
				println("Failed to get next page:", err)
				return
			}

			if page == nil {
				break
			}

			println(page.Title())
		}
	}
}
