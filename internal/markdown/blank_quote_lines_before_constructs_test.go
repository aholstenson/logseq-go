package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// A quote can have a blank `>` line between a paragraph and a list, math
// block or other construct. Saving the block keeps the blank `>` line.
var _ = Describe("Parsing then outputting blank quote lines before constructs", func() {
	// Given a blank quote line before a list, When written, Then the
	// blank line stays.
	FullyEqual("Blank quote line before a list", "- > a\n  >\n  > * b")

	// Given a blank quote line before a math block, When written, Then
	// the blank line stays.
	FullyEqual("Blank quote line before a math block", "- > a\n  >\n  > $$\n  > x\n  > $$")

	// Given a list right after a paragraph in a quote, When written, Then
	// the text is the same. This works now and must keep working.
	FullyEqual("List right after a paragraph in a quote", "- > a\n  > * b")
})
