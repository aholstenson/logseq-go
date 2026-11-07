package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// A quote marker can have no space after it (`>q`) or be next to a second
// marker (`>>`). Saving the block keeps the quote markers as they are in
// the file.
var _ = Describe("Parsing then outputting quote markers without a space", func() {
	// Given quote lines without a space after the marker, When written,
	// Then no space is added.
	FullyEqual("Quote without a space", "- >q\n  >r")

	// Given a nested quote without spaces, When written, Then no space is
	// added between the markers.
	FullyEqual("Nested quote without spaces", "- >> q")

	// Given a quote line without a space under the text of a block, When
	// written, Then no space is added.
	FullyEqual("Quote without a space under the text of a block", "- a\n  >x")
})
