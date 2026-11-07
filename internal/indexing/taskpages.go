package indexing

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
)

// Option changes how the index reads pages and blocks, for settings that the
// config of the graph does not have.
type Option func(*BlugeIndex)

// WithTaskPages sets whether the task marker and the priority of a block link
// the block to the pages with the same name, so that `TODO [#A] x` links to
// the pages `TODO` and `A`. Logseq does this, so it is on by default.
func WithTaskPages(enabled bool) Option {
	return func(i *BlugeIndex) {
		i.taskPagesDisabled = !enabled
	}
}

// WithNamespaceParentRefs sets whether a reference to a page in a namespace
// also links the block to the namespaces above it, so that `[[a/b/c]]` links
// to `a/b` and `a` as well. Logseq does this, so it is on by default.
func WithNamespaceParentRefs(enabled bool) Option {
	return func(i *BlugeIndex) {
		i.namespaceParentRefsDisabled = !enabled
	}
}

// taskPageTitles returns the titles of the pages that the task marker and the
// priority of the block link to, normalized with normalizeRef. Logseq reads
// the marker and the priority only at the start of the block, as
// content.Block.TaskStatus and content.Block.Priority do.
func (i *BlugeIndex) taskPageTitles(block *content.Block) []string {
	if i.taskPagesDisabled {
		return nil
	}

	titles := make([]string, 0, 2)
	if status := block.TaskStatus(); status != content.TaskStatusNone {
		titles = append(titles, normalizeRef(status.String()))
	}

	// Logseq drops a blank title, such as the one of the priority `[# ]`
	if title := normalizeRef(priorityValue(block.Priority())); strings.TrimSpace(title) != "" {
		titles = append(titles, title)
	}

	return titles
}
