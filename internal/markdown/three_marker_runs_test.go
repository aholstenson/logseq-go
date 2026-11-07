package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// Strikethrough can be written with three tildes on each side, as in
// `~~~a~~~`. Saving the block keeps three tildes and does not write four.
var _ = Describe("Parsing then outputting strikethrough with three markers", func() {
	// Given strikethrough with three tildes, When written, Then the text is
	// the same.
	FullyEqual("Strikethrough with three tildes", "- x ~~~a~~~")
})
