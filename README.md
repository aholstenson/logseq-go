# logseq-go

logseq-go is a Go library to work with a [Logseq](https://logseq.com) graph,
with support for reading and modifying journals and pages.

⚠️ **Note:** This library is still in early development, it may destroy your data
when pages are modified. Please open issues if you find any bugs.

## Features

- Read and write journals and pages
- Rename and delete pages, with references to a renamed page updated across the
  graph
- Search, linked references and lookup of blocks by their id
- Aliases and namespaces, such as `Parent/Child`
- Rich content model
  - Blocks
  - Formatting via headings, paragraphs, lists, code blocks, etc.
  - Highlights via `^^Example^^`
  - Tables in the GitHub Flavored Markdown syntax
  - LaTeX via `$formula$` and math blocks
  - Footnotes via `[^1]` and `[^1]: The footnote`
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

In a graph with indexing enabled, opening a page by one of the aliases in its
`alias::` property opens the page the alias belongs to, instead of creating a
second page for the same content.

The properties that hold a list of pages, such as `alias::` and `tags::`, have
their values read as references to those pages, so `tags:: Book, Reading`
points at two pages without either being written as a link. The graph decides
which other properties are read that way via `:property/separated-by-commas`,
and which properties never point at a page via
`:ignored-page-references-keywords`.

The blocks that reference a page, what Logseq shows as its linked references,
are available from the page itself:

```go
references, err := page.LinkedReferences(ctx)

for _, reference := range references.Results() {
  block, referencingPage, err := reference.Open()
  // ...
}
```

Pages with a `/` in their title are part of a namespace, and the pages in a
namespace can be found from the page it belongs to:

```go
page, err := graph.OpenPage("Parent/Child")

page.Namespace() // Parent

children, err := page.NamespaceChildren(ctx)
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

Pages in the namespace of a renamed page keep their titles, unless the rename
asks for them to come along:

```go
// Renames Old, Old/Child and Old/Child/Grandchild
err = tx.RenamePage(ctx, "Old", "New", logseq.WithNamespaceChildren())
```

Deleted pages are removed from the graph, unless the graph is opened with
`logseq.WithRecycleDeletedPages()`, in which case they are moved to the
`logseq/.recycle` directory that Logseq recovers deleted pages from.

### Tasks

Blocks have helpers to read and change tasks:

```go
block.SetTaskStatus(graph.Workflow().Todo())
block.SetPriority(content.PriorityA)
block.SetDeadline(content.NewDeadline(time.Now().AddDate(0, 0, 7)))

// Marks the task as done in the same way as Logseq. A task with a repeating
// date stays open, its date moves to the next occurrence and the logbook
// records that it was done.
block.CompleteTask(time.Now())
```

### Revisions and conflicts

Every page has a revision, which is a hash of the content of its file. A page
that has no file yet has the empty revision. Saving a transaction fails with
`logseq.ErrConflict` if a page it writes has changed since it was opened, and
nothing is written.

An editor that shows a page for some time can save on top of the revision it
started from. The save then also fails if the page changed while the editor
showed it:

```go
page, err := graph.OpenPage("Example")
revision := page.Revision()

// Later, when the user saves
tx := graph.NewTransaction()
page, err = tx.OpenPage("Example")
// Apply the changes of the user to page

err = tx.Save(logseq.WithExpectedRevision(page, revision))
if errors.Is(err, logseq.ErrConflict) {
  // The page changed, reload it and let the user merge their changes
}
```

### Watching for changes

`graph.Watch()` sends an event when a page is changed, renamed or deleted on
disk. `PageUpdated` has the revision of the page. Saves made via the graph also
cause events, and these have `OwnWrite` set, so that they can be told apart
from changes made by Logseq or other programs.

A rename is a removed file and a created file on disk. When the graph has an
index and the created file has the same content as the removed file had, the
two are sent as one `PageRenamed` event. Otherwise they are sent as
`PageDeleted` and `PageUpdated`.

When the graph can not watch for changes, or can not read or index a changed
file, a `WatchError` event is sent. It has the path of the file, when the error
is for one file:

```go
watcher := graph.Watch()
defer watcher.Close()

for {
  select {
  case e := <-watcher.Events():
    switch event := e.(type) {
    case *logseq.PageUpdated:
      if event.OwnWrite {
        continue
      }

      fmt.Println("Changed outside:", event.Page.Title(), event.Revision)
    case *logseq.PageRenamed:
      fmt.Println("Renamed:", event.OldTitle, "to", event.Page.Title())
    case *logseq.PageDeleted:
      fmt.Println("Deleted:", event.Title)
    case *logseq.WatchError:
      fmt.Println("Error:", event)
    }
  case <-watcher.Done():
    return
  }
}
```

## Navigating the graph

`logseq-go` provides the ability to index the graph to allow it to be navigated.

In memory indexing or persistent indexing can be used:

```go
// In-memory index that rebuilds on every open
graph, err := logseq.Open(ctx, "path/to/graph", logseq.WithInMemoryIndex())

// Persistent index that is saved to disk, is loaded and refreshed on open
graph, err := logseq.Open(ctx, "path/to/graph", logseq.WithIndex("path/to/index"))
```

Opening a graph with an index can take some time, as pages that changed since
the last open are indexed again. Use `WithListener` to show the progress:

```go
graph, err := logseq.Open(ctx, "path/to/graph",
  logseq.WithIndex("path/to/index"),
  logseq.WithListener(func(event logseq.OpenEvent) {
    switch e := event.(type) {
    case *logseq.ScanStarted:
      // The graph looks for pages that changed, the total is not known yet
    case *logseq.SyncStarted:
      // e.Total pages will be indexed, e.Unchanged are up to date
    case *logseq.PageIndexed:
      // e.Done of e.Total pages are handled
    case *logseq.PageIndexFailed:
      // A page could not be read and was skipped, e.Err has the reason
    case *logseq.SyncCompleted:
      // The index is up to date, e.Removed pages were removed as their
      // files were deleted while the graph was closed
    case *logseq.OpenCompleted:
      // The graph is ready to use
    }
  }),
)
```

The events are sent in the order above. A page that can not be read does not
stop the graph from opening. It is reported with `PageIndexFailed` and is tried
again the next time the graph is opened. A graph without an index sends only
`OpenCompleted`.

Pages can be searched for by title, content, references, tags, etc:

```go
pages, err := graph.SearchPages(ctx, logseq.WithQuery(logseq.TitleMatches("Example")))

for _, r := range pages.Results() {
  // Pages can be opened from the result
  page, err := r.Open()
  ...
}
```

Blocks can also be searched:

```go
blocks, err := graph.SearchBlocks(ctx, logseq.WithQuery(logseq.ContentMatches("Example")))

for _, r := range blocks.Results() {
  // Blocks can be opened from the result
  block, page, err := r.Open()
  ...
}
```

Tasks can be found by their status, priority and dates:

```go
// Open tasks with priority A that have a deadline before the end of today
now := time.Now()
endOfToday := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.Local)
tasks, err := graph.SearchBlocks(ctx, logseq.WithQuery(logseq.And(
  logseq.IsOpenTask(),
  logseq.PriorityIs(content.PriorityA),
  logseq.DeadlineBetween(time.Time{}, endOfToday),
)))
```

## Limitations

This library is limited to working with Markdown files. As the library provides
an AST for the content there might be some issues with formatting that comes
out wrong after having been read and saved again.

If this happens to you, please do open an issue with an example of content
that is causing the issue.

## License

This project is licensed under the MIT license, see [LICENSE](LICENSE).
