package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// A footnote definition can be in a quote, and a quote can be in a footnote
// definition. A lazy line of the inner one keeps the marker or the
// indentation of the outer one. Saving the block keeps each line as it is in
// the file.
var _ = Describe("Parsing then outputting lazy lines of footnotes in quotes and quotes in footnotes", func() {
	// Given a footnote definition in a quote with a lazy line that has the
	// quote marker, When written, Then the line keeps the quote marker.
	FullyEqual("Lazy footnote line in a quote", "> [^1]: a\n> b")

	// Given a quote in a footnote definition with a lazy line that has the
	// footnote indentation, When written, Then the line keeps the
	// indentation.
	FullyEqual("Lazy quote line in a footnote", "x[^1]\n\n[^1]: > a\n    b")
})
