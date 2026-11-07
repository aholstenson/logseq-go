package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// A custom drawer, such as `:foo:` ... `:END:`, can have a blank line before
// it. Saving the block keeps the blank line.
var _ = Describe("Parsing then outputting a blank line before a custom drawer", func() {
	// Given a blank line before a custom drawer in a block, When written,
	// Then the blank line stays.
	FullyEqual("Blank line before a custom drawer", "- t\n\n  :foo:\n  x\n  :END:")

	// Given a blank line before a logbook in a block, When written, Then
	// the text is the same. This works now and must keep working.
	FullyEqual("Blank line before a logbook", "- t\n\n  :LOGBOOK:\n  :END:")
})
