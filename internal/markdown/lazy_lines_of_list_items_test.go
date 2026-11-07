package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// A line after a list item can have less indentation than the text of the
// item. CommonMark and Logseq read it as a lazy line of the item. Saving
// the block keeps the line at its column.
var _ = Describe("Parsing then outputting lazy lines of list items", func() {
	// Given a lazy line at the column of the block text after a star
	// item, When written, Then the line keeps its column.
	FullyEqual("Lazy line at the block text column after a star item", "- a\n  * b\n  c")

	// Given a lazy line at the column of the block text after an ordered
	// item, When written, Then the line keeps its column.
	FullyEqual("Lazy line at the block text column after an ordered item", "- a\n  1. one\n  2. two\n  more")
})
