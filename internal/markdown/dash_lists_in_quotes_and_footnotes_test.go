package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// A list with the `-` marker can be in a quote or in a footnote
// definition, where a `-` line does not start a block. Saving the block keeps
// the `-` marker and does not change it to `*`.
var _ = Describe("Parsing then outputting dash lists in quotes and footnotes", func() {
	// Given a dash list in a quote on the bullet line, When written, Then
	// the markers stay `-`.
	FullyEqual("Dash list in a quote in a block", "- > - a\n  > - b")

	// Given a dash list in a footnote definition, When written, Then the
	// marker stays `-`.
	FullyEqual("Dash list in a footnote", "x[^1]\n\n[^1]: - a")

	// Given a dash list in a quote under the text of a block, When written,
	// Then the markers stay `-`.
	FullyEqual("Dash list in a quote under the text of a block", "- a\n  > - c\n  > - d")

	// Given a dash item in a quote on the bullet line, When written, Then
	// the marker stays `-`.
	FullyEqual("Dash item in a quote on the bullet line", "- > - x")

	// Given a dash item in a quote under the text of a block, When written,
	// Then the marker stays `-`.
	FullyEqual("Dash item in a quote under the text of a block", "- a\n  > - x")

	// Given a star list in a quote, When written, Then the text is the
	// same. This works now and must keep working.
	FullyEqual("Star list in a quote in a block", "- > * a\n  > * b")
})
