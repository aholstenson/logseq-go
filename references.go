package logseq

import (
	"context"
	"fmt"
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/indexing"
)

// pageTitlesEqual checks if two titles refer to the same page. Logseq does not
// distinguish between titles that only differ in case, so `[[example]]` and
// `[[Example]]` point at the same page.
func pageTitlesEqual(a string, b string) bool {
	return strings.EqualFold(a, b)
}

// retargetPageReferences points every reference to the page `from` in the given
// tree at the page `to`, returning the number of references that changed.
func retargetPageReferences(root content.HasChildren, from string, to string) int {
	changed := 0
	for _, node := range root.Children().FilterDeep(content.IsPageReference()) {
		ref := node.(content.PageRef)
		if !pageTitlesEqual(ref.GetTo(), from) {
			continue
		}

		ref.SetTo(to)
		changed++
	}

	return changed
}

// referencingSubPaths finds the sub paths of all the pages that reference the
// page with the given title. Requires indexing to be enabled.
func (g *Graph) referencingSubPaths(ctx context.Context, title string) ([]string, error) {
	if g.index == nil {
		return nil, fmt.Errorf("indexing is not enabled")
	}

	// References are indexed per block, so the blocks are collected in batches
	// and reduced to the pages they belong to.
	const batchSize = 500

	subPaths := make([]string, 0)
	seen := make(map[string]struct{})

	for from := 0; ; from += batchSize {
		results, err := g.index.SearchBlocks(ctx, indexing.References(title), indexing.SearchOptions{
			Size: batchSize,
			From: from,
		})
		if err != nil {
			return nil, err
		}

		if results.Size() == 0 {
			break
		}

		for _, block := range results.Results() {
			if _, ok := seen[block.PageSubPath]; ok {
				continue
			}

			seen[block.PageSubPath] = struct{}{}
			subPaths = append(subPaths, block.PageSubPath)
		}

		if from+results.Size() >= results.Count() {
			break
		}
	}

	return subPaths, nil
}
