package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// A property line or a SCHEDULED line under the text of a block can be
// indented with a tab. Saving the block keeps the tab.
var _ = Describe("Parsing then outputting property and date lines after a tab", func() {
	// Given a property line after a tab, When written, Then the tab stays.
	FullyEqual("Tab before a property line", "- x\n\tfoo:: bar")

	// Given a SCHEDULED line after a tab, When written, Then the tab stays.
	FullyEqual("Tab before a SCHEDULED line", "- TODO x\n\tSCHEDULED: <2024-01-01 Mon>")
})
