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

	ctx := context.Background()

	indexOpt := logseq.WithInMemoryIndex()
	if indexDirectory != "" {
		indexOpt = logseq.WithIndex(indexDirectory)
	}

	graph, err := logseq.Open(ctx, directory, indexOpt, logseq.WithListener(func(event logseq.OpenEvent) {
		switch e := event.(type) {
		case *logseq.ScanStarted:
			println("Looking for changed pages...")
		case *logseq.SyncStarted:
			fmt.Printf("Indexing %d pages, %d are up to date\n", e.Total, e.Unchanged)
		case *logseq.PageIndexed:
			fmt.Printf("[%d/%d] %s\n", e.Done, e.Total, e.SubPath)
		case *logseq.SyncCompleted:
			fmt.Printf("Indexed %d pages and removed %d in %s\n", e.Indexed, e.Removed, e.Duration)
		case *logseq.OpenCompleted:
			println("Graph is open")
		}
	}))
	if err != nil {
		println("Failed to open graph:", err.Error())
		return
	}
	defer graph.Close()

	println("Ready to search for blocks. Type 'exit' or 'quit' to exit.")

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
		blocks, err := graph.SearchBlocks(ctx, logseq.WithQuery(logseq.ContentMatches(query)))
		if err != nil {
			println("Failed to search blocks:", err.Error())
			return
		}

		if blocks.Size() < blocks.Count() {
			println("Showing", blocks.Size(), "of", blocks.Count(), "results")
		} else {
			println("Showing", blocks.Size(), "results")
		}
		println("")

		for _, page := range blocks.Results() {
			switch page.PageType() {
			case logseq.PageTypeDedicated:
				println("📝 " + page.PageTitle())
			case logseq.PageTypeJournal:
				println("📅 " + page.PageTitle())
			}
			println(page.Preview())
			println("------")
		}
	}
}
