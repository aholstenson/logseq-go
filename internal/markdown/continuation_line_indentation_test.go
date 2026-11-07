package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// Logseq keeps the spaces at the start of a continuation line as text. Saving
// a page keeps them, also when they are more than the indentation of the
// block.
var _ = Describe("Parsing then outputting continuation lines with indentation", func() {
	// Given page text with an indented second line, When written, Then the
	// second line keeps its spaces.
	FullyEqual("Indented line of page text", "text\n  more")

	// Given page text with a second line indented with a tab, When written,
	// Then the second line keeps its tab.
	FullyEqual("Tab indented line of page text", "text\n\tmore")

	// Given a block with a line indented more than the block, When written,
	// Then the line keeps its extra spaces.
	FullyEqual("Line indented more than its block", "- a\n  b\n     more")

	// Given emphasis over two lines, When written, Then the second line keeps
	// its spaces.
	FullyEqual("Indented line in emphasis", "**a\n  b**")

	// Given a quote with spaces after the marker on its second line, When
	// written, Then the spaces stay.
	FullyEqual("Indented line in a quote", "- a\n  > q\n  >   r")

	// Given an indented paragraph with a second line indented more, When
	// written, Then each line keeps its indentation.
	FullyEqual("Line indented more than an indented paragraph", "- a\n\n      line 1\n        line 2")

	// Given page text with an indented SCHEDULED line, When written, Then the
	// SCHEDULED line keeps its spaces.
	FullyEqual("Indented SCHEDULED line after page text", "text\n  SCHEDULED: <2024-03-04 Mon>")

	// Given a task with a SCHEDULED line indented more than the block, When
	// written, Then the SCHEDULED line keeps its extra spaces.
	FullyEqual("SCHEDULED line indented more than its block", "- TODO a\n   SCHEDULED: <2024-03-04 Mon>")

	// Given a task with a normal SCHEDULED line, When written, Then no spaces
	// are added.
	FullyEqual("SCHEDULED line with the indentation of its block", "- TODO a\n  SCHEDULED: <2024-03-04 Mon>")
})
