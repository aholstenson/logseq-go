package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// Saving a page that was not changed by the user keeps its whitespace: blank
// lines, trailing spaces, empty block markers and line prefixes.
var _ = Describe("Parsing then outputting whitespace in blocks", func() {
	// Given a block with a logbook and text after it, When written, Then no
	// blank line is added after the logbook.
	FullyEqual("Text after a logbook",
		"- DONE x\n  :LOGBOOK:\n  CLOCK: [2024-03-04 Mon 09:00:12]\n  :END:\n  text")

	// Given blank lines between blocks, When written, Then they are kept.
	FullyEqual("Blank line between a block and its child", "- a\n\n\t- b")
	FullyEqual("Blank line between two blocks", "- a\n\n- b")
	FullyEqual("Blank line between two child blocks", "- a\n\t- b\n\n\t- c")
	FullyEqual("Blank line after the text of a block, before the next block", "- a\n\n  more\n\n- b")

	// Given an empty block written as "-", When written, Then no space is
	// added after the bullet.
	FullyEqual("Empty block between two blocks", "- a\n-\n- b")
	FullyEqual("Empty block with spaces after the bullet", "- a\n-  \n- b")
	FullyEqual("Empty block with a child", "-\n\t- c")

	// Given an empty block with an id property, When written, Then the
	// property stays on its own line.
	FullyEqual("Empty block with an id property", "- \n  id:: 6650a7b2-0000-4000-8000-000000000001")

	// Given property lines with special whitespace, When written, Then the
	// lines are the same.
	FullyEqual("Property with an empty value and no space after the name in a block", "- a\n  b::")
	FullyEqual("Property with an empty value and spaces after the name", "- a\n  b::   \n  c:: d")
	FullyEqual("Property with an empty value and a tab after the name", "- a\n  b::\t")
	FullyEqual("Property with trailing spaces after its value", "- a\n  b:: c  ")
	FullyEqual("Property with trailing spaces before the next property", "- a\n  b:: c \n  d:: e\t")
	FullyEqual("Property with trailing spaces before text", "- a\n  b:: c \n  text")
	FullyEqual("Directive with trailing spaces", "#+title: x \n\n- a")
	FullyEqual("Directive with an empty value and a space", "#+title: \n\n- a")
	FullyEqual("Line that looks like a property and ends with two spaces", "- a\n  #x::  \n  b")

	// Given a quote with a lazy continuation line, When written, Then no
	// quote marker is added to the second line.
	FullyEqual("Quote with a lazy continuation line", "- > a\n  b")
	FullyEqual("Quote with a lazy line and a marked line", "- > a\n  b\n  > c")
	FullyEqual("Nested quote with a lazy line", "- > > a\n  b")
	FullyEqual("Quote with a lazy line at the top level", "> a\nb")
	FullyEqual("Quote with a lazy line before an id", "- > a\n  b\n  id:: 6650a7b2-0000-4000-8000-000000000001")
})
