package utils

import "github.com/aholstenson/logseq-go/content"

// LocationOf returns the place of a block on its page: its position among the
// blocks next to it at each level, from the top of the page. This is the same
// as the Location of a block found in the index. Returns false if the block is
// not on a page.
func LocationOf(block *content.Block) ([]int, bool) {
	var location []int
	current := block
	for {
		parent, ok := current.Parent().(*content.Block)
		if !ok {
			// The block is the root of a page, or not on a page.
			return nil, false
		}

		idx := 0
		for sibling := current.PreviousSibling(); sibling != nil; sibling = sibling.PreviousSibling() {
			if _, ok := sibling.(*content.Block); ok {
				idx++
			}
		}
		location = append(location, idx)

		// The root of the page is the only block without a parent.
		if parent.Parent() == nil {
			break
		}
		current = parent
	}

	for i, j := 0, len(location)-1; i < j; i, j = i+1, j-1 {
		location[i], location[j] = location[j], location[i]
	}
	return location, true
}
