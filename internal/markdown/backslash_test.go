package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq keeps the raw text of a block in the file. A backslash that the
// user typed must stay in the file when a page is saved, and a backslash
// that stops markup (such as `\#`) must not turn into live markup.
var _ = Describe("Backslashes in saved text", func() {
	pageRefTitles := func(block *content.Block) []string {
		titles := []string{}
		for _, ref := range block.Children().PageReferences() {
			titles = append(titles, ref.(content.PageRef).GetTo())
		}
		return titles
	}

	writeAndReadAgain := func(input string) (string, *content.Block) {
		block, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())

		written, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())

		again, err := markdown.ParseString(written)
		Expect(err).ToNot(HaveOccurred())
		return written, again
	}

	Describe("plain text", func() {
		It("keeps the backslashes of a Windows path in the text and in the file", func() {
			// Given a block with a Windows path
			input := `- C:\Users\foo`

			// When it is parsed
			block, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			// Then the text keeps both backslashes
			text := block.Children().FindDeep(content.IsOfType[*content.Text]())
			Expect(text).ToNot(BeNil())
			Expect(text.(*content.Text).Value).To(Equal(`C:\Users\foo`))

			// And when it is written, Then the file text is unchanged
			written, err := markdown.AsString(block)
			Expect(err).ToNot(HaveOccurred())
			Expect(written).To(Equal(input))
		})

		It("keeps an escaped backslash when the block is saved", func() {
			// Given a block with `\\`, which Logseq shows as one backslash
			input := `- a \\ b`

			// When it is parsed and written
			written := parseAndOutput(input)

			// Then the file text is unchanged and the backslashes are not removed
			Expect(written).ToNot(Equal("- a  b"))
			Expect(written).To(Equal(input))
		})

		It("keeps the backslashes of a regular expression when the block is saved", func() {
			// Given a block with a regular expression, When it is written
			input := `- regex \d+`

			// Then the file text is unchanged
			Expect(parseAndOutput(input)).To(Equal(input))
		})
	})

	Describe("escaped markup", func() {
		It("does not turn an escaped hash into a tag when the block is saved", func() {
			// Given a block with `\#tag`, which Logseq does not read as a tag
			input := `- \#tag`

			// When it is written and read again
			written, again := writeAndReadAgain(input)

			// Then the file text is unchanged and there is no page ref `tag`
			Expect(written).To(Equal(input))
			Expect(pageRefTitles(again)).To(BeEmpty())
		})

		It("does not turn an escaped block ref into a block ref when the block is saved", func() {
			// Given a block with an escaped block ref
			input := `- \((6571b1a6-1111-4111-8111-111111111111))`

			// When it is written and read again
			written, again := writeAndReadAgain(input)

			// Then the file text is unchanged and there is no block ref
			Expect(written).To(Equal(input))
			Expect(again.Children().FilterDeep(content.IsOfType[*content.BlockRef]())).To(BeEmpty())
		})
	})

	Describe("page links", func() {
		It("keeps a backslash in the title of a page link", func() {
			// Given a block with `[[a\b]]`, which Logseq reads as the page `a\b`
			input := `- [[a\b]]`

			// When it is parsed
			block, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			// Then the page ref is `a\b`
			Expect(pageRefTitles(block)).To(Equal([]string{`a\b`}))

			// And when it is written, Then the file text is unchanged
			written, err := markdown.AsString(block)
			Expect(err).ToNot(HaveOccurred())
			Expect(written).To(Equal(input))
		})

		It("does not add a backslash before a single closing bracket in a page link", func() {
			// Given a block with `[[a]b]]`, which Logseq reads as the page `a]b`
			input := `- [[a]b]]`

			// When it is written, Then the file text is unchanged
			Expect(parseAndOutput(input)).To(Equal(input))
		})
	})

	Describe("property values", func() {
		It("keeps the backslashes of a Windows path in a property value", func() {
			// Given a block with a property value that is a Windows path
			input := "- a\n  path:: C:\\Users\\x"

			// When it is parsed
			block, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			// Then the property value keeps the backslashes
			value := block.Blocks()[0].Properties().Get("path")
			text := value.FindDeep(content.IsOfType[*content.Text]())
			Expect(text).ToNot(BeNil())
			Expect(text.(*content.Text).Value).To(Equal(`C:\Users\x`))

			// And when it is written, Then the file text is unchanged
			written, err := markdown.AsString(block)
			Expect(err).ToNot(HaveOccurred())
			Expect(written).To(Equal(input))
		})

		It("keeps an escaped comma in a tags property value", func() {
			// Given a tags value with an escaped comma, When it is written
			input := "- a\n  tags:: x\\, y"

			// Then the file text is unchanged (it is not split into two tags)
			Expect(parseAndOutput(input)).To(Equal(input))
		})
	})
})
