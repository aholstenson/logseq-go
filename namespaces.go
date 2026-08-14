package logseq

import (
	"context"
	"fmt"

	"github.com/aholstenson/logseq-go/internal/indexing"
)

// pageTitlesUnderNamespace finds the titles of all the pages in a namespace,
// including the ones deeper in it. Requires indexing to be enabled.
func (g *Graph) pageTitlesUnderNamespace(ctx context.Context, namespace string) ([]string, error) {
	if g.index == nil {
		return nil, fmt.Errorf("indexing is not enabled")
	}

	titles := make([]string, 0)

	err := eachResult(func(opts indexing.SearchOptions) (indexing.SearchResults[*indexing.Page], error) {
		return g.index.SearchPages(ctx, indexing.UnderNamespace(namespace), opts)
	}, func(page *indexing.Page) {
		if page.Type == indexing.PageTypeDedicated && page.Title != "" {
			titles = append(titles, page.Title)
		}
	})
	if err != nil {
		return nil, err
	}

	return titles, nil
}

// namespaceRenameTarget returns the title a page in a namespace gets when the
// namespace is renamed, which is its own title with the part that is the old
// namespace replaced by the new one. The second return value is false for
// titles that are not in the namespace.
func namespaceRenameTarget(title string, from string, to string) (string, bool) {
	if len(title) <= len(from) || title[len(from)] != '/' {
		return "", false
	}

	if !pageTitlesEqual(title[:len(from)], from) {
		return "", false
	}

	return to + title[len(from):], true
}
