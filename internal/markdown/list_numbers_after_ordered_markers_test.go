package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// An ordered list can start on the line of an item of another ordered
// list, as in `2. 1. a`. Saving keeps the number of the inner item and does
// not use the number of the outer item.
var _ = Describe("Parsing then outputting ordered lists on the line of an ordered item", func() {
	// Given an inner number 1 after an outer number 2, When written, Then
	// the inner number stays 1.
	FullyEqual("Inner number after an outer ordered marker", "2. 1. a")

	// Given an inner number 1 after an outer number 10, When written, Then
	// the inner number stays 1.
	FullyEqual("Inner number after an outer number ten", "10. 1. a")

	// Given an inner list that starts at 9 on the line of an ordered item
	// in a block, When written, Then the inner number stays 9.
	FullyEqual("Inner list that starts at nine in a block", "- t\n  1. 9. a\n     10. b")
})
