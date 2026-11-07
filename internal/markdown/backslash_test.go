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

// A backslash before ASCII punctuation, as in `\*`, is read as that character,
// the same as Logseq does. The backslash is not needed in all places, but it
// is in the file, so it must stay there when a page is saved.
var _ = Describe("Backslashes that are not needed to stop markup", func() {
	// Given a block with a backslash that is not needed, When it is parsed
	// and written, Then the text is the same as the input.
	FullyEqual("Escaped asterisk between spaces", `- a \* b`)
	FullyEqual("Escaped full stop", `- a \. b`)
	FullyEqual("Escaped parenthesis", `- a \( b`)
	FullyEqual("Escaped hash before a space", `- a \# b`)
	FullyEqual("Escaped hash at the end of the text", `- x \#`)
	FullyEqual("Escaped hash marks at the end of a heading", `- ## x \##`)
	FullyEqual("Escaped asterisk after a task marker", `- TODO a \* b`)
	FullyEqual("Escaped asterisk in bold text", `- **a \* b**`)
	FullyEqual("Escaped characters on two lines of a block", "- a \\*\n  b \\_ c")
	FullyEqual("Escaped opening asterisk of text that looks like emphasis", `- a \*b*`)
	FullyEqual("Escaped opening asterisk at the start of a block", `- \*a*`)
	FullyEqual("Escaped opening underscore of text that looks like emphasis", `- a \_b_`)
	FullyEqual("Escaped opening tilde of text that looks like strikethrough", `- \~~a~~`)
	FullyEqual("Escaped opening equal sign of text that looks like a highlight", `- \==a==`)
	FullyEqual("Escaped asterisk before emphasis in the same line", `- a \*b* *c*`)

	It("still escapes both asterisks of text made with the API", func() {
		// Given a block with text that would read back as emphasis
		page := content.NewBlock(
			content.NewBlock(content.NewParagraph(content.NewText("a *b*"))),
		)

		// When it is written
		v, err := markdown.AsString(page)
		Expect(err).ToNot(HaveOccurred())

		// Then both asterisks are escaped
		Expect(v).To(Equal(`- a \*b\*`))
	})

	FullyEqual("Escaped asterisk in a page link", `- [[a \* b]]`)
	FullyEqual("Escaped asterisk in a page link with a label", `- [x]([[a \* b]])`)
	FullyEqual("Escaped asterisk in a page link with a label and a title", `- [x]([[a \* b]] "t")`)
	FullyEqual("Escaped asterisk in a nested page link", `- [[x [[a \* b]]]]`)
	FullyEqual("Escaped asterisk in an embedded page link", `- ![[a \* b]]`)
	FullyEqual("Escaped asterisk in a page link in a property value", `- k:: [[a \* b]]`)

	It("reads a page link with an escaped character as a ref to the page without the backslash", func() {
		// Given a block with a page link with an escaped asterisk
		root, err := markdown.ParseString(`- [[a \* b]]`)
		Expect(err).ToNot(HaveOccurred())

		// Then the link points at the page `a * b`, the same as Logseq
		refs := root.Blocks()[0].Children().PageReferences()
		Expect(refs).To(HaveLen(1))
		Expect(refs[0].(content.PageRef).GetTo()).To(Equal("a * b"))
	})

	It("writes a page link whose target was changed with the new target", func() {
		// Given a page link with an escaped asterisk, whose target is then
		// changed, as when the page is renamed
		root, err := markdown.ParseString(`- [[a \* b]]`)
		Expect(err).ToNot(HaveOccurred())
		links := root.Children().FilterDeep(content.IsOfType[*content.PageLink]())
		Expect(links).To(HaveLen(1))
		links[0].(*content.PageLink).To = "c"

		// When it is written
		v, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())

		// Then the link has the new target
		Expect(v).To(Equal("- [[c]]"))
	})

	It("reads an escaped character without its backslash, the same as Logseq", func() {
		// Given a block with an escaped asterisk
		root, err := markdown.ParseString(`- a \* b`)
		Expect(err).ToNot(HaveOccurred())

		// Then the text has the asterisk and no backslash
		text, ok := root.Blocks()[0].Content()[0].(*content.Paragraph).FirstChild().(*content.Text)
		Expect(ok).To(BeTrue(), "first node of the paragraph is text")
		Expect(text.Value).To(Equal("a * b"))
	})

	It("writes a text that was changed after it was read without the old backslashes", func() {
		// Given a block with an escaped asterisk, whose text is then changed
		root, err := markdown.ParseString(`- a \* b`)
		Expect(err).ToNot(HaveOccurred())
		text := root.Blocks()[0].Content()[0].(*content.Paragraph).FirstChild().(*content.Text)
		text.Value = "c * d"

		// When it is written
		v, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())

		// Then the new text is written as it is
		Expect(v).To(Equal("- c * d"))
	})
})

// The `#` marks before a tag are text, as in "##x", which is the text `#`
// and the tag `#x`. The text must not get a backslash when it is written.
var _ = Describe("Hash marks directly before a tag", func() {
	FullyEqual("Hash mark before a tag at the start of a block", "- ##x")
	FullyEqual("Hash mark before a tag after text", "- x ##y")

	It("keeps the tag after the hash mark", func() {
		// Given a block whose text starts with a hash mark and a tag
		root, err := markdown.ParseString("- ##x")
		Expect(err).ToNot(HaveOccurred())

		// Then the block has a ref to the page of the tag
		refs := root.Blocks()[0].Children().PageReferences()
		Expect(refs).To(HaveLen(1))
		Expect(refs[0].(content.PageRef).GetTo()).To(Equal("x"))
	})
})
