package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// An empty quote is written as a line with only `>`, and an empty code block
// as a fence line and a closing fence line with no line between them. They
// must read back as the same nodes, and must not join with the node after
// them.
var _ = Describe("Empty quotes and code blocks", func() {
	write := func(node content.Node) string {
		v, err := markdown.AsString(node)
		Expect(err).ToNot(HaveOccurred())
		return v
	}

	paragraph := func(value string) *content.Paragraph {
		return content.NewParagraph(content.NewText(value))
	}

	It("keeps a paragraph after an empty quote at the start of a block apart", func() {
		// Given a block of an empty quote and the paragraph `b`
		root := content.NewBlock(content.NewBlock(content.NewBlockquote(), paragraph("b")))

		// When it is written
		output := write(root)

		// Then the quote is a line of its own
		Expect(output).To(Equal("- >\n\n  b"))

		// And it reads back as an empty quote and a paragraph
		again, err := markdown.ParseString(output)
		Expect(err).ToNot(HaveOccurred())
		block := again.Blocks()[0]
		quote, ok := block.FirstChild().(*content.Blockquote)
		Expect(ok).To(BeTrue(), "output: %q", output)
		Expect(quote.FirstChild()).To(BeNil(), "output: %q", output)
		Expect(block.FirstChild().NextSibling()).To(BeAssignableToTypeOf(&content.Paragraph{}), "output: %q", output)
	})

	DescribeTable("writes the same text after it is read again",
		func(node content.Node, expected string) {
			// Given a block with the node
			root := content.NewBlock(content.NewBlock(node))

			// When it is written, read and written again
			output := write(root)
			again, err := markdown.ParseString(output)
			Expect(err).ToNot(HaveOccurred())

			// Then both outputs are the expected text
			Expect(output).To(Equal(expected))
			Expect(write(again)).To(Equal(expected))
		},
		Entry("an empty code block", content.NewCodeBlock(""), "- ```\n  ```"),
		Entry("an empty code block in a quote", content.NewBlockquote(content.NewCodeBlock("")), "- > ```\n  > ```"),
		Entry("an empty code block in a list item", content.NewUnorderedList(content.NewListItem(content.NewCodeBlock(""))), "- * ```\n    ```"),
		Entry("an empty code block in a footnote", content.NewFootnoteDefinition("1", content.NewCodeBlock("")), "- [^1]: ```\n      ```"),
		Entry("an empty quote", content.NewBlockquote(), "- >"),
		Entry("an empty quote in a quote", content.NewBlockquote(content.NewBlockquote()), "- > >"),
		Entry("an empty quote in a list item", content.NewUnorderedList(content.NewListItem(content.NewBlockquote())), "- * >"),
		Entry("an empty quote in a footnote", content.NewFootnoteDefinition("1", content.NewBlockquote()), "- [^1]: >"),
	)
})
