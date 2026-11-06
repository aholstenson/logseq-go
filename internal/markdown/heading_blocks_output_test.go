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
