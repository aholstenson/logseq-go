package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// The text of a footnote definition can start on the line after the label.
// Saving the block keeps the text on that line.
var _ = Describe("Parsing then outputting the layout of footnote definitions", func() {
	// Given a footnote definition with an empty first line, When written,
	// Then the text stays on the next line.
	FullyEqual("Footnote definition with an empty first line", "x[^1]\n\n[^1]:\n    a")
})
