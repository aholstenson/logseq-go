package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// A blank line in a block can have spaces, and a thematic break can be
// indented more than the text of the block. Saving the block keeps these
// lines as they are in the file.
var _ = Describe("Parsing then outputting spaces on blank lines and before thematic breaks", func() {
	// Given a blank line with spaces in a block, When written, Then the
	// spaces stay.
	FullyEqual("Blank line with spaces in a block", "- a\n   \n  b")

	// Given a thematic break that is indented one column more than the
	// text of the block, When written, Then its indentation stays.
	FullyEqual("Indented thematic break", "- a\n\n   ---")
})
