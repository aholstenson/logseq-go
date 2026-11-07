package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// A list item can have only its marker on the first line, and its text on
// the next line. Saving the block keeps the text on its own line.
var _ = Describe("Parsing then outputting list items with an empty first line", func() {
	// Given a star item with its text on the next line, When written,
	// Then the text stays on the next line.
	FullyEqual("Star item with an empty first line", "- *\n    a")
})
