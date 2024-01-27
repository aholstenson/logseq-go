# logseq-go

logseq-go is a Go library to work with a [Logseq](https://logseq.com) graph,
with support for reading and modifying journals and pages.

⚠️ **Note:** This library is still in early development, it may destroy your data
when pages are modified. Please open issues if you find any bugs.

## Usage

```go
graph, err := logseq.Open("path/to/graph")

// Open journal for today and add a block to the end of it
today, err := graph.Journal(time.Now())

today.AddBlock(content.NewBlock(
  content.NewText("Hello!")
))

err = today.Save()
```

## Limitations

This library is limited to working with Markdown files. As the library provides
an AST for the content there might be some issues with formatting that comes
out wrong after having been read and saved again.

If this happens to you, please do open an issue with an example of content
that is causing the issue.
