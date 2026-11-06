package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq splits macro arguments at commas, but for `query` and `embed` it
// joins them again, so the full text between the macro name and `}}` is the
// query or the embed target. Journal titles such as `Feb 28th, 2026` have a
// comma, so this is common.
var _ = Describe("Commas in query and embed macros", func() {
	parseFirst := func(input string) (*content.Block, content.Node) {
		block, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		paragraph := block.Blocks()[0].Children()[0]
		Expect(paragraph).To(BeAssignableToTypeOf(&content.Paragraph{}))
		return block, paragraph.(*content.Paragraph).Children()[0]
	}

	Describe("query", func() {
		It("reads and saves a query with a comma in a property filter", func() {
			// Given a query with a comma
			input := "- {{query (property type, book)}}"

			// When it is parsed
			block, node := parseFirst(input)

			// Then the query has the full text
			Expect(node).To(BeAssignableToTypeOf(&content.Query{}))
			Expect(node.(*content.Query).Query).To(Equal("(property type, book)"))

			// And when it is written, Then the file text is unchanged
			written, err := markdown.AsString(block)
			Expect(err).ToNot(HaveOccurred())
			Expect(written).To(Equal(input))
		})

		It("reads and saves a query that refers to a journal title with a comma", func() {
			// Given a query that refers to a journal page
			input := "- {{query (and [[Feb 28th, 2026]] (task TODO))}}"

			// When it is parsed
			block, node := parseFirst(input)

			// Then the query has the full text
			Expect(node).To(BeAssignableToTypeOf(&content.Query{}))
			Expect(node.(*content.Query).Query).To(Equal("(and [[Feb 28th, 2026]] (task TODO))"))

			// And when it is written, Then the file text is unchanged
			written, err := markdown.AsString(block)
			Expect(err).ToNot(HaveOccurred())
			Expect(written).To(Equal(input))
		})
	})

	Describe("query in quotes", func() {
		It("keeps the quotes of a full-text query", func() {
			// Given a full-text query in quotes
			input := "- {{query \"x\"}}"

			// When it is parsed
			block, node := parseFirst(input)

			// Then the query has the quotes, the same as in Logseq
			Expect(node).To(BeAssignableToTypeOf(&content.Query{}))
			Expect(node.(*content.Query).Query).To(Equal(`"x"`))

			// And when it is written, Then the file text is unchanged
			written, err := markdown.AsString(block)
			Expect(err).ToNot(HaveOccurred())
			Expect(written).To(Equal(input))
		})
	})

	Describe("embed", func() {
		It("keeps a second embed argument when the block is saved", func() {
			// Given an embed with a second argument, When it is written
			input := "- {{embed [[p]], extra}}"

			// Then the second argument is kept
			Expect(parseAndOutput(input)).To(Equal(input))
		})

		It("keeps an embed of two page links as a macro", func() {
			// Given an embed with two page links
			input := "- {{embed [[a]], [[b]]}}"

			// When it is parsed
			_, node := parseFirst(input)

			// Then it is not an embed of a page named "a]], [[b"
			Expect(node).To(BeAssignableToTypeOf(&content.Macro{}))

			// And when it is written, Then the text is unchanged
			Expect(parseAndOutput(input)).To(Equal(input))
		})

		It("reads an embed of a journal title with a comma as a page embed", func() {
			// Given an embed of a journal page
			input := "- {{embed [[Feb 28th, 2026]]}}"

			// When it is parsed
			_, node := parseFirst(input)

			// Then it is an embed of the page `Feb 28th, 2026`
			Expect(node).To(BeAssignableToTypeOf(&content.PageEmbed{}))
			Expect(node.(*content.PageEmbed).To).To(Equal("Feb 28th, 2026"))
		})

		It("writes an embed of a journal title with a comma without quotes", func() {
			// Given a page embed made with the API for a title with a comma
			block := content.NewBlock(content.NewParagraph(content.NewPageEmbed("Feb 28th, 2026")))

			// When it is written
			written, err := markdown.AsString(block)
			Expect(err).ToNot(HaveOccurred())

			// Then Logseq can read the embed target (no quotes around it)
			Expect(written).ToNot(ContainSubstring(`"`))
			Expect(written).To(Equal("{{embed [[Feb 28th, 2026]]}}"))
		})
	})
})
