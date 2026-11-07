package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// A line in a nested quote can have fewer `>` markers than the quote has
// levels, as `b` has in "> > a\n> b". CommonMark and Logseq read such a line
// as a lazy line of the paragraph in the inner quote. Saving the block keeps
// the markers of the line as they are in the file.
var _ = Describe("Parsing then outputting lazy lines of nested quotes", func() {
	// Given a nested quote whose second line has one marker, When written,
	// Then the line does not get a second marker.
	FullyEqual("Lazy line with one marker in a nested quote", "- > > a\n  > b")

	// Given a nested quote whose two lines have two markers, When written,
	// Then the text is the same. This works now and must keep working.
	FullyEqual("Two lines with two markers in a nested quote", "- > > a\n  > > b")
})
