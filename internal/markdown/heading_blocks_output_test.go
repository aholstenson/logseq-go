package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq stores a heading block as a bullet followed by the heading marks,
// for example "- ## Section". These tests check that the heading marks stay
// on the bullet line when a page is written back.
var _ = Describe("Parsing then outputting heading blocks", func() {
	// Given a bullet whose first line is a heading, When the page is parsed
	// and written, Then the text is the same as the input.
	FullyEqual("Heading block of level 1 with a page link", "- # heading [[h]]")
	FullyEqual("Heading block of level 2", "- ## Heading")
	FullyEqual("Heading block followed by a second block", "- # Title\n- next")
	FullyEqual("Heading block nested under a block", "- a\n\t- ## Sub\n\t\t- c")
	FullyEqual("Heading block with a page link nested under a block", "- parent\n\t- ### Sub [[ref]]")
	FullyEqual("Heading block with a property and a child", "- # H\n  collapsed:: true\n\t- c")
	FullyEqual("Heading block with an id property", "- ### h3\n  id:: 6571b1a6-1111-4111-8111-111111111111")
	FullyEqual("Heading block with a tab indented child", "- # h\n\t- child")
	FullyEqual("Heading block with a task marker after the heading marks", "- ## TODO Heading task")

	// Given a heading with `#` marks at the end of its line, When written,
	// Then the marks are kept. Logseq reads them as text of the heading,
	// while CommonMark removes them.
	FullyEqual("Heading block with closing heading marks", "- ## x ##")
	FullyEqual("Heading block with only heading marks after the first marks", "- ### ###")
	FullyEqual("Heading block with heading marks inside and at the end", "- # x # y #")
	FullyEqual("Heading task block with closing heading marks", "- ## TODO x ##")
	FullyEqual("Heading line of a page with closing heading marks", "## x ##")

	// Given a heading with more than one space after its marks, or spaces at
	// the end of its line, When written, Then the spaces are kept.
	FullyEqual("Heading block without text and a space after the marks", "- ## ")
	FullyEqual("Heading block without text and two spaces after the marks", "- ##  ")
	FullyEqual("Heading block with three spaces after the marks", "- ##   x")
	FullyEqual("Heading block with a tab after the marks", "- ##\tx")
	FullyEqual("Heading block with a space at the end", "- ## x ")
	FullyEqual("Heading block with spaces around its text and closing marks", "- ##   x  ##  ")
	FullyEqual("Heading block with a space at the end and a child", "- ## x \n\t- c")
	FullyEqual("Heading task block without text and a space at the end", "- ## TODO ")
	FullyEqual("Heading line of a page with a space at the end", "## x ")
	FullyEqualWhenReadWith("Heading line of a page with a space before its marks", " # b\n- a", markdown.AsPage())
	FullyEqual("Heading line with a space before its marks after a block", "- a\n # b")

	It("reads the text of a heading without the spaces after its marks, the same as Logseq", func() {
		// Given a heading block with three spaces after its marks
		root, err := markdown.ParseString("- ##   x")
		Expect(err).ToNot(HaveOccurred())

		// Then the text of the heading is `x`
		heading, ok := root.Blocks()[0].Content()[0].(*content.Heading)
		Expect(ok).To(BeTrue(), "first node of the block is a heading")
		text, ok := heading.FirstChild().(*content.Text)
		Expect(ok).To(BeTrue(), "first node of the heading is text")
		Expect(text.Value).To(Equal("x"))
	})

	It("reads closing heading marks as text of the heading, the same as Logseq", func() {
		// Given a heading block with `#` marks at the end of its line
		root, err := markdown.ParseString("- ## x ##")
		Expect(err).ToNot(HaveOccurred())

		// Then the text of the heading holds the marks
		Expect(root.Blocks()).To(HaveLen(1))
		heading, ok := root.Blocks()[0].Content()[0].(*content.Heading)
		Expect(ok).To(BeTrue(), "first node of the block is a heading")
		Expect(heading.Level).To(Equal(2))
		text, ok := heading.FirstChild().(*content.Text)
		Expect(ok).To(BeTrue(), "first node of the heading is text")
		Expect(text.Value).To(Equal("x ##"))
	})

	// Given a heading block with text on the next line, When written, Then
	// the first line keeps its heading marks.
	It("keeps the heading marks of a heading block with text below it", func() {
		v := parseAndOutput("- ## Heading\n  body")
		Expect(v).To(HavePrefix("- ## Heading\n"))
	})

	It("reads a written heading block with text below it as a heading again", func() {
		v := parseAndOutput("- # H\n  more")

		block, err := markdown.ParseString(v)
		Expect(err).ToNot(HaveOccurred())
		Expect(block.Blocks()).To(HaveLen(1))
		heading, ok := block.Blocks()[0].Content()[0].(*content.Heading)
		Expect(ok).To(BeTrue(), "first node of the block is a heading, written text: %q", v)
		Expect(heading.Level).To(Equal(1))
	})

	// Given a heading block that is made with the API, When written, Then
	// the line is a bullet followed by the heading marks.
	It("writes a heading block made with the API with its heading marks", func() {
		page := content.NewBlock(
			content.NewBlock(content.NewHeading(2, content.NewText("x"))),
		)

		v, err := markdown.AsString(page)
		Expect(err).ToNot(HaveOccurred())
		Expect(v).To(Equal("- ## x"))
	})
})

// Logseq starts a new block at each heading line, also when the line is
// inside the text of a block. The heading block is a child of the block.
var _ = Describe("A heading line inside a block", func() {
	parse := func(input string) *content.Block {
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(root.Blocks()).To(HaveLen(1))
		return root.Blocks()[0]
	}

	It("starts a child block, the same as Logseq", func() {
		// Given a block with a heading on its second line
		block := parse("- x\n  ## y")

		// Then the heading is the first node of a child block
		Expect(block.Blocks()).To(HaveLen(1))
		heading, ok := block.Blocks()[0].Content()[0].(*content.Heading)
		Expect(ok).To(BeTrue(), "first node of the child block is a heading")
		Expect(heading.Level).To(Equal(2))

		// And the heading is not part of the text of the parent block
		for _, node := range block.Content() {
			_, isHeading := node.(*content.Heading)
			Expect(isHeading).To(BeFalse(), "parent block has no heading")
		}
	})

	It("reads the task marker of the heading as a task of the child block", func() {
		// Given a block with a heading task on its second line
		block := parse("- x\n  ## TODO y")

		// Then the parent block is not a task
		Expect(block.TaskStatus()).To(Equal(content.TaskStatusNone))

		// And its child block is a TODO task
		Expect(block.Blocks()).ToNot(BeEmpty())
		Expect(block.Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusTodo))
	})

	It("makes the heading a sibling of the child blocks that follow it", func() {
		// Given a heading line and then a child bullet at the same indentation
		block := parse("- x\n  ## TODO y\n  - child")

		// Then the parent block has two child blocks: the heading and the bullet
		Expect(block.Blocks()).To(HaveLen(2))
		Expect(block.Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusTodo))
	})

	It("does not add a blank line before the heading when written", func() {
		// Given a block with a heading on its second line
		// When it is parsed and written
		v := parseAndOutput("- x\n  ## y")

		// Then no blank line is added between the text and the heading
		Expect(v).ToNot(ContainSubstring("\n\n"))
	})
})

// Logseq reads `-` directly followed by heading marks, as in "-## Title", as
// a bullet with a heading.
var _ = Describe("A heading right after the bullet dash", func() {
	It("is a heading block, the same as Logseq", func() {
		// Given a page whose first line has no space between `-` and `#`
		root, err := markdown.ParseString("-# x")
		Expect(err).ToNot(HaveOccurred())

		// Then the line is a block with a level 1 heading
		Expect(root.Blocks()).To(HaveLen(1))
		heading, ok := root.Blocks()[0].Content()[0].(*content.Heading)
		Expect(ok).To(BeTrue(), "first node of the block is a heading")
		Expect(heading.Level).To(Equal(1))
	})

	It("is a task block when a marker follows the heading marks", func() {
		// Given a heading task without a space after `-`, and a next block
		root, err := markdown.ParseString("-## TODO x\n- y")
		Expect(err).ToNot(HaveOccurred())

		// Then both lines are blocks, and the first block is a TODO task
		Expect(root.Blocks()).To(HaveLen(2))
		Expect(root.Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusTodo))

		// And the line is not page content
		Expect(root.TaskStatus()).To(Equal(content.TaskStatusNone))
	})

	It("is the next block after a block with text", func() {
		// Given a block, and then a heading bullet without a space
		root, err := markdown.ParseString("- a\n-## b")
		Expect(err).ToNot(HaveOccurred())

		// Then the heading bullet is a second top-level block
		Expect(root.Blocks()).To(HaveLen(2))
		Expect(root.Blocks()[0].Blocks()).To(BeEmpty())
	})

	It("is a child block when indented under a block", func() {
		// Given a heading bullet without a space, indented with a tab
		root, err := markdown.ParseString("- a\n\t-## TODO b")
		Expect(err).ToNot(HaveOccurred())

		// Then it is a child block, and a TODO task
		Expect(root.Blocks()).To(HaveLen(1))
		Expect(root.Blocks()[0].Blocks()).To(HaveLen(1))
		Expect(root.Blocks()[0].Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusTodo))
	})

	It("is not a block when the heading marks have no space after them", func() {
		// Given a line where text follows the `#` marks directly
		root, err := markdown.ParseString("- a\n-##b")
		Expect(err).ToNot(HaveOccurred())

		// Then the line is text of the block, the same as Logseq
		Expect(root.Blocks()).To(HaveLen(1))
		Expect(root.Blocks()[0].Blocks()).To(BeEmpty())
	})

	It("is more text of a quote right before it, the same as Logseq", func() {
		// Given a quote, and then a heading bullet without a space
		root, err := markdown.ParseString("- > q\n-## b")
		Expect(err).ToNot(HaveOccurred())

		// Then the line is part of the quote and not a second block
		Expect(root.Blocks()).To(HaveLen(1))
	})

	It("is written as a bullet with a space and reads back the same", func() {
		// Given a heading bullet without a space, When it is written
		v := parseAndOutput("-## TODO x\n- y")

		// Then the bullet has a space, the same as Logseq writes it
		Expect(v).To(Equal("- ## TODO x\n- y"))
	})
})

// Logseq reads any number of `#` marks before a task marker, also more than
// the six levels that Markdown headings have.
var _ = Describe("A task marker after more than six heading marks", func() {
	firstBlock := func(input string) *content.Block {
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(root.Blocks()).ToNot(BeEmpty())
		return root.Blocks()[0]
	}

	It("reads the marker after seven marks, the same as Logseq", func() {
		// Given a block that starts with seven `#` marks and a marker
		// Then the block is a TODO task
		Expect(firstBlock("- ####### TODO x").TaskStatus()).To(Equal(content.TaskStatusTodo))
	})

	It("reads the marker and priority after eight marks", func() {
		// Given a block that starts with eight `#` marks, a marker and a priority
		block := firstBlock("- ######## DONE [#A] x")

		// Then the block is a DONE task with priority A
		Expect(block.TaskStatus()).To(Equal(content.TaskStatusDone))
		Expect(block.Priority()).To(Equal(content.PriorityA))
	})

	It("does not change the text when written", func() {
		// Given a block that starts with seven `#` marks and a marker
		// When it is parsed and written
		// Then the text is the same as the input
		Expect(parseAndOutput("- ####### TODO x")).To(Equal("- ####### TODO x"))
	})

	It("reads a heading task on a later line as a child block", func() {
		// Given a block with seven `#` marks and a marker on its second line
		block := firstBlock("- a\n  ####### TODO y")

		// Then the parent block is not a task, and its child block is
		Expect(block.TaskStatus()).To(Equal(content.TaskStatusNone))
		Expect(block.Blocks()).To(HaveLen(1))
		Expect(block.Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusTodo))
	})

	It("does not read a heading when text follows the marks directly", func() {
		// Given a block where text follows seven `#` marks without a space
		block := firstBlock("- #######TODO x")

		// Then the block is not a task, and starts with no heading
		Expect(block.TaskStatus()).To(Equal(content.TaskStatusNone))
		_, isHeading := block.Content()[0].(*content.Heading)
		Expect(isHeading).To(BeFalse())
	})

	It("keeps `#` marks at the end of the line when written", func() {
		// Given a heading with seven marks and `#` marks after its text
		// When it is parsed and written
		// Then the text is the same as the input
		Expect(parseAndOutput("- ####### x ##")).To(Equal("- ####### x ##"))
	})

	It("escapes seven `#` marks in text made with the API", func() {
		// Given a block with a paragraph whose text starts with seven marks
		page := content.NewBlock(
			content.NewBlock(content.NewParagraph(content.NewText("####### x"))),
		)

		// When it is written and read again
		v, err := markdown.AsString(page)
		Expect(err).ToNot(HaveOccurred())
		root, err := markdown.ParseString(v)
		Expect(err).ToNot(HaveOccurred())

		// Then the text is still a paragraph and not a heading
		Expect(root.Blocks()).To(HaveLen(1))
		_, isParagraph := root.Blocks()[0].Content()[0].(*content.Paragraph)
		Expect(isParagraph).To(BeTrue(), "written text: %q", v)
	})
})
