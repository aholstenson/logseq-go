package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	"github.com/aholstenson/logseq-go/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq has no link reference definitions. A line such as `[Status]: done`
// is text that Logseq shows, so it must be read as text and saved.
var _ = Describe("Lines that look like link reference definitions", func() {
	It("reads a label line as text and saves it", func() {
		// Given a block `[Status]: done`
		input := "- [Status]: done"

		// When it is parsed
		block, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())

		// Then the block has a paragraph with the text
		Expect(block).To(tests.EqualNode(content.NewBlock(
			content.NewBlock(
				content.NewParagraph(content.NewText("[Status]: done")),
			),
		)))

		// And when it is written, Then the file text is unchanged
		written, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())
		Expect(written).To(Equal(input))
	})

	It("saves a numbered citation line", func() {
		// Given a block `[1]: https://example.com/source`, When it is written
		input := "- [1]: https://example.com/source"

		// Then the file text is unchanged
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("saves a citation line with a title after a paragraph", func() {
		// Given a paragraph and a citation line with a title, When it is written
		input := "- a\n\n  [1]: http://x.com \"T\""

		// Then the file text is unchanged
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("saves a reference style link and its definition as written", func() {
		// Given a reference style link and its definition, When it is written
		input := "- [ref][1]\n\n  [1]: http://x.com"

		// Then the file text is unchanged
		Expect(parseAndOutput(input)).To(Equal(input))
	})
})
