package markdown

import (
	"github.com/aholstenson/logseq-go/content"
)

// blockLevels holds the level that Logseq gives each block of a page. The
// level of a bullet is one more than the number of spaces and tabs before it,
// see lineIndentCharacters. A heading line that is not in a bullet, and the
// content before the first bullet, have level 1.
type blockLevels map[*content.Block]int

// nestByLevels moves the blocks of a page to the parents that Logseq gives
// them. Goldmark nests bullets by columns, where a tab is up to four columns.
// Logseq counts characters, and nests each bullet by comparing its level with
// the levels of the bullets before it:
//
//   - The same level as the last bullet: a sibling of that bullet.
//   - A higher level: a child of the last bullet.
//   - A lower level that one of the parents of the last bullet has: a sibling
//     of that parent.
//   - A lower level that no parent has: a child of the last parent with a
//     lower level. The bullet then takes the level of the bullet that comes
//     after that parent, for the bullets that follow it.
//
// Thus "c" is a sibling of "b" in "- a\n  - b\n\t- c", as the tab is one
// character and the two spaces are two.
//
// If a block has no level, the page has blocks that are not read from
// bullets, and the tree is kept as it is.
func nestByLevels(page *content.Block, levels blockLevels) {
	preBlock, blocks, ok := blocksInFileOrder(page, levels)
	if !ok {
		return
	}

	type parentEntry struct {
		block *content.Block
		level int
	}

	// parents are the last block of each level, from the page down to the
	// last block. Their levels always increase.
	parents := []parentEntry{{block: page, level: 0}}

	// rootContent is the content before the first bullet when it is not in a
	// pre-block yet. Logseq reads it as a block of level 1, so a bullet with
	// a higher level after it is its child.
	var rootContent *content.Block
	if preBlock != nil {
		parents = append(parents, parentEntry{block: preBlock, level: 1})
	} else if len(page.Content()) > 0 {
		rootContent = content.NewPreBlock()
		parents = append(parents, parentEntry{block: rootContent, level: 1})
	}

	for _, block := range blocks {
		level := levels[block]
		for {
			last := parents[len(parents)-1]
			if level == last.level {
				parents[len(parents)-1] = parentEntry{block: block, level: level}
				parents[len(parents)-2].block.AddChild(block)
				break
			}

			if level > last.level {
				last.block.AddChild(block)
				parents = append(parents, parentEntry{block: block, level: level})
				break
			}

			// keep is how many parents have a level that is not higher than
			// the level of the block.
			keep := 0
			found := false
			for _, parent := range parents {
				if parent.level > level {
					break
				}

				found = parent.level == level
				keep++
			}

			if found {
				// Go back to the parent with the same level, and read the
				// block again as its sibling.
				parents = parents[:keep]
				continue
			}

			// The block takes the level of the parent after the one that it
			// is a child of.
			level = parents[keep].level
			parents[keep-1].block.AddChild(block)
			parents = append(parents[:keep], parentEntry{block: block, level: level})
			break
		}
	}

	if rootContent != nil && len(rootContent.Blocks()) > 0 {
		// Bullets are indented under the content before the first bullet, so
		// that content is a pre-block with these children.
		children := rootContent.Blocks()
		for _, node := range page.Content() {
			page.RemoveChild(node)
			rootContent.AddChild(node)
		}

		for _, child := range children {
			rootContent.RemoveChild(child)
			rootContent.AddChild(child)
		}

		page.PrependChild(rootContent)
	}
}

// blocksInFileOrder takes all blocks out of the page, and gives them back in
// the order they have in the file, without their child blocks. A pre-block
// with content is kept in the page and is given back on its own. A pre-block
// without content is removed, as Logseq does not have it. Returns false, and
// changes nothing, if a block has no level.
func blocksInFileOrder(page *content.Block, levels blockLevels) (*content.Block, []*content.Block, bool) {
	var preBlock *content.Block
	var blocks []*content.Block
	var collect func(parent *content.Block) bool
	collect = func(parent *content.Block) bool {
		for _, block := range parent.Blocks() {
			if block.IsPreBlock() && parent == page && preBlock == nil && len(blocks) == 0 {
				if len(block.Content()) > 0 {
					preBlock = block
				}
			} else if _, ok := levels[block]; ok {
				blocks = append(blocks, block)
			} else {
				return false
			}

			if !collect(block) {
				return false
			}
		}

		return true
	}

	if !collect(page) {
		return nil, nil, false
	}

	for _, block := range append(blocks, preBlock) {
		if block == nil {
			continue
		}

		for _, child := range block.Blocks() {
			block.RemoveChild(child)
		}
	}

	for _, block := range page.Blocks() {
		if block != preBlock {
			page.RemoveChild(block)
		}
	}

	return preBlock, blocks, true
}

// set keeps the level of a block. It does nothing if levels is nil.
func (l blockLevels) set(block *content.Block, level int) {
	if l != nil {
		l[block] = level
	}
}
