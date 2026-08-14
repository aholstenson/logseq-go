# logseq-go

logseq-go is a Go library to work with a [Logseq](https://logseq.com) graph,
with support for reading and modifying journals and pages.

⚠️ **Note:** This library is still in early development, it may destroy your data
when pages are modified. Please open issues if you find any bugs.

## Features

- Read and write journals and pages
- Rename and delete pages, with references to a renamed page updated across the
  graph
- Rich content model
  - Blocks
  - Formatting via headings, paragraphs, lists, code blocks, etc.
  - Page links via `[[Example]]`
  - Tags via `#Example` and `#[[Example with space]]`
  - Macros via `{{macro param1 param2}}`
  - Block references via `((block-id))`

## Usage

Open a graph to access its content:

```go
graph, err := logseq.Open(ctx, "path/to/graph")
```

Content can be opened read only:

```go
journalPage, err := graph.OpenJournal(time.Now())
page, err := graph.OpenPage("Example")

for _, block := range page.Blocks() {
  // ...
}
```

Blocks that have an id, which is what block references such as `((id))` point
at, can be opened directly in a graph that has indexing enabled:

```go
block, page, err := graph.OpenBlock(ctx, "65a1b2c3-d4e5-6789-abcd-ef0123456789")
```

Content can also be opened for writing, by creating a transaction:

```go
tx := graph.NewTransaction()

today, err := tx.OpenJournal(time.Now())

today.AddBlock(content.NewBlock(
  content.NewText("Hello!")
))

// Save all the changes made
err = tx.Save()
```

Pages can be renamed and deleted in a transaction as well. Renaming points all
the references to the page, such as `[[Old]]` and `#Old`, at the new title,
which requires the graph to be opened with indexing enabled:

```go
tx := graph.NewTransaction()

err = tx.RenamePage(ctx, "Old", "New")
err = tx.DeletePage("Unwanted")

err = tx.Save()
```

Deleted pages are removed from the graph, unless the graph is opened with
`logseq.WithRecycleDeletedPages()`, in which case they are moved to the
`logseq/.recycle` directory that Logseq recovers deleted pages from.

## Limitations

This library is limited to working with Markdown files. As the library provides
an AST for the content there might be some issues with formatting that comes
out wrong after having been read and saved again.

If this happens to you, please do open an issue with an example of content
that is causing the issue.

## License

This project is licensed under the MIT license, see [LICENSE](LICENSE).