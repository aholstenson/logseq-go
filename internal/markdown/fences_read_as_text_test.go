package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// Some fence lines do not open a code block for the parser, for example a
// fence whose closing line is outside the list item, or a fence on the first
// line of a footnote definition. The lines are then text with a code span
// over more than one line. Saving the block keeps the fence lines and does
// not join them into inline code on one line.
var _ = Describe("Parsing then outputting fence lines that are read as text", func() {
	// Given a fence in a list item that is closed at the column of the
	// block text, When written, Then the three lines stay.
	FullyEqual("Fence closed at the block column under a list item", "- * a\n    ```\n    x\n  ```")

	// Given a fence on the first line of a footnote definition, When
	// written, Then the three lines stay.
	FullyEqual("Fence on the first line of a footnote definition", "x[^1]\n\n[^1]: ```\n    x\n    ```")

	// Given a code span with two backticks over two lines, When written,
	// Then the backticks stay.
	FullyEqual("Code span over two lines keeps its backticks", "- a ``x\n  y`` b")
})
