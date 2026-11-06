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
