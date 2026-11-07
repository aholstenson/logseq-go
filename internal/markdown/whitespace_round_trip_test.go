package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

	// Given a block whose last line ends with spaces or a tab, When parsed
	// and written, Then the spaces are kept.
	FullyEqual("Space at the end of a block", "- x ")
	FullyEqual("Tab at the end of a block before the next block", "- x\t\n- y")
	FullyEqual("Spaces at the end of a block before a property", "- x  \n  id:: 6650a7b2-0000-4000-8000-000000000001")
	FullyEqual("Space at the end of a block before a child", "- x \n\t- c")
	FullyEqual("Space after bold text at the end of a block", "- **b** ")
	FullyEqual("Space after a page link at the end of a block", "- [[p]] ")
	FullyEqual("Space after a page link on the second line of a block", "- x\n  [[p]] ")
	FullyEqual("Space at the end of text before the first bullet", "x \n- y")
	FullyEqual("Space after a task marker and text", "- TODO x ")
	FullyEqual("Two spaces after a task marker without text", "- TODO  ")
	FullyEqual("Space after a task priority", "- TODO [#A] ")
	FullyEqual("Escaped backslash before a space at the end of a block", `- x\\ `)

	// Given a paragraph whose first line is indented further than the block
	// or page that holds it, When parsed and written, Then the indentation is
	// kept.
	FullyEqual("Indented text after a heading line after a block", "- a\n## b\n  text")
	FullyEqualWhenReadWith("Indented text after a heading line of a page", "- a\n## b\n  text", markdown.AsPage())
	FullyEqualWhenReadWith("Indented text before the first bullet of a page", "  text\n- a", markdown.AsPage())
	FullyEqual("Text indented one more space after a blank line in a block", "- a\n\n   b")
	FullyEqual("Indented text after properties", "- a\n  k:: v\n   t")
	FullyEqual("Indented text that starts with a page link after properties", "- a\n  b:: c\n   [[p]] t")
	FullyEqualWhenReadWith("Indented text after a directive", "#+title: x\n  text\n- a", markdown.AsPage())
	FullyEqual("Quote with spaces before its text", "- >   q")
	FullyEqual("Task with a space before its marker", " TODO Task")

	// Given a bullet with more than one space, or a tab, before its text,
	// When parsed and written, Then the spaces are kept.
	FullyEqual("Bullet with two spaces before its text", "-  x")
	FullyEqual("Bullet with three spaces before its text", "-   x")
	FullyEqual("Bullet with four spaces before its text", "-    x")
	FullyEqual("Bullet with a tab before its text", "-\tx")
	FullyEqual("Bullet with a space and a tab before its text", "- \tx")
	FullyEqual("Bullet with three spaces before a heading", "-   ## x")
	FullyEqual("Bullet with three spaces before a task", "-   TODO x")
	FullyEqual("Child bullet with two spaces before its text", "- a\n\t-  b")
	FullyEqual("Bullet with three spaces before its text and a child", "-   x\n\t- c")
	FullyEqual("Bullet with three spaces before its text and an id", "-   x\n  id:: 6650a7b2-0000-4000-8000-000000000001")

	// Given a block with a line indented less than its text, When parsed and
	// written, Then the line keeps its indentation. Logseq reads the line as a
	// part of the block.
	FullyEqual("Line without indentation in a block", "- x\ny")
	FullyEqual("Line indented one space in a block", "- x\n y")
	FullyEqual("Two lines without indentation in a block", "- x\ny\nz")
	FullyEqual("Line without indentation before the next block", "- x\ny\n- b")
	FullyEqual("Line without indentation before an id", "- x\ny\n  id:: 6650a7b2-0000-4000-8000-000000000001")
	FullyEqual("Line without indentation after a hard line break", "- x\\\ny")
	FullyEqual("Line with a tab and no more indentation in a child block", "- a\n\t- x\n\ty")
	FullyEqual("Line with a tab and one space in a child block", "- a\n\t- x\n\t y")

	// Given a bullet with more than one space before its text and a line
	// below it, When parsed and written, Then the line keeps its indentation.
	FullyEqual("Line indented two spaces under a bullet with three spaces", "-   x\n  y")
	FullyEqual("Line indented three spaces under a bullet with three spaces", "-   x\n   y")
	FullyEqual("Line indented four spaces under a bullet with three spaces", "-   x\n    y")
	FullyEqual("Paragraph indented four spaces under a bullet with three spaces", "-   x\n\n    b")

	// Given a list or a footnote with more than one space, or a tab, after
	// its marker, When parsed and written, Then the spaces are kept.
	FullyEqual("List item with two spaces after its marker", "- x\n  * a\n  *  b")
	FullyEqual("Ordered list item with two spaces after its number", "- x\n  1.  a")
	FullyEqual("List item with a tab after its marker", "- x\n  +\ta")
	FullyEqual("List item with two spaces after its marker and a second line", "- x\n  *  a\n     b")
	FullyEqual("Ordered list item with two spaces after its number and a second line", "- x\n  1.  a\n      b")
	FullyEqual("Footnote definition with two spaces after its label", "[^1]:  x")
	FullyEqual("Footnote definition with a tab after its label", "[^1]:\tx")
	FullyEqual("Footnote definition with three spaces after its label in a block", "- a[^1]\n  [^1]:   x")
	FullyEqual("Footnote definition without text", "[^1]:")

	// Given an ordered list whose numbers are not counted up from one, When
	// parsed and written, Then the numbers are kept.
	FullyEqual("Ordered list with the same number for each item", "- x\n  1. a\n  1. b")
	FullyEqual("Ordered list that starts after one", "- x\n\n  3. a\n  4. b")
	FullyEqual("Ordered list with a number with a leading zero", "- x\n  01. a")

	// Given a list with a blank line between two items, When parsed and
	// written, Then the blank line is kept.
	FullyEqual("Blank line between two items of an ordered list", "- x\n  1. a\n\n  2. b")
	FullyEqual("Blank line between two items of a list", "- x\n  * a\n\n  * b")
	FullyEqual("Blank line between the first two of three items", "- x\n  * a\n\n  * b\n  * c")
	FullyEqual("Blank line between two items of a nested list", "- x\n  * a\n    * b\n\n    * c")
	FullyEqual("Blank line between two items of a list at the top level", "* a\n\n* b")
	FullyEqual("Blank line between two items of a list in a quote", "> * a\n>\n> * b")

	// Given a quote with two paragraphs and a line with only `>` between
	// them, When parsed and written, Then the paragraphs stay apart.
	FullyEqual("Two paragraphs in a quote", "> a\n>\n> b")
	FullyEqual("Two paragraphs in a quote in a block", "- > a\n  >\n  > b")
	FullyEqual("Two paragraphs in a nested quote", "> > a\n> >\n> > b")
})

var _ = Describe("Writing lists made with the API", func() {
	It("numbers the items of an ordered list from one", func() {
		// Given an ordered list with two items and no numbers
		list := content.NewList(content.ListTypeOrdered,
			content.NewListItem(content.NewText("a")),
			content.NewListItem(content.NewText("b")),
		)

		// When it is written
		v, err := markdown.AsString(content.NewBlock(content.NewBlock(list)))
		Expect(err).ToNot(HaveOccurred())

		// Then the items are numbered from one, with one space after each
		Expect(v).To(Equal("- 1. a\n  2. b"))
	})
})
