package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// A quote can have two heading lines next to each other. Saving the block
// does not add a blank `>` line between them.
var _ = Describe("Parsing then outputting headings in quotes", func() {
	// Given two heading lines in a quote, When written, Then no blank
	// quote line is added.
	FullyEqual("Two heading lines in a quote", "- > # a\n  > ## b")
})
