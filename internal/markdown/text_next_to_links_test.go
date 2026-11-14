package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A text next to a link or a footnote reference can join with it into other
// syntax: a `!` before a link makes an image, and a `(` after a footnote
// reference makes a link. The writer must escape such text.
var _ = Describe("Text next to a link or a footnote reference", func() {
	write := func(node content.Node) string {
		v, err := markdown.AsString(node)
		Expect(err).ToNot(HaveOccurred())
		return v
	}

	readBack := func(output string) *content.Block {
		root, err := markdown.ParseString(output)
		Expect(err).ToNot(HaveOccurred())
		return root
	}

	It("does not make an image from `!` before a link", func() {
		// Given a paragraph of the text `a!` and a link
		paragraph := content.NewParagraph(content.NewText("a!"), content.NewLink("https://a.com", content.NewText("l")))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then it has the link and no image
		Expect(root.Children().FindDeep(content.IsOfType[*content.Image]())).To(BeNil(), "output: %q", output)
		Expect(root.Children().FindDeep(content.IsOfType[*content.Link]())).ToNot(BeNil(), "output: %q", output)
	})

	It("does not make an image from `!` before a block reference with a label", func() {
		// Given a paragraph of the text `!` and a block reference with a label
		paragraph := content.NewParagraph(content.NewText("!"), content.NewBlockRef("6512c4c4-0000-0000-0000-000000000000", content.NewText("l")))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then it has the block reference and no image
		Expect(root.Children().FindDeep(content.IsOfType[*content.Image]())).To(BeNil(), "output: %q", output)
		Expect(root.Children().FindDeep(content.IsOfType[*content.BlockRef]())).ToNot(BeNil(), "output: %q", output)
	})

	It("does not escape `!` before a link on the next line", func() {
		// Given a paragraph of the line `a!` and a link on the next line
		text := content.NewText("a!")
		text.SoftLineBreak = true
		paragraph := content.NewParagraph(text, content.NewLink("https://a.com", content.NewText("l")))

		// When it is written, Then the `!` is not escaped
		Expect(write(paragraph)).To(Equal("a!\n[l](https://a.com)"))
	})

	It("does not make a link from a footnote reference and `(`", func() {
		// Given a paragraph of a footnote reference and the text `(a)`
		paragraph := content.NewParagraph(content.NewFootnoteRef("1"), content.NewText("(a)"))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then it has the footnote reference and no link
		Expect(root.Children().FindDeep(content.IsOfType[*content.Link]())).To(BeNil(), "output: %q", output)
		Expect(root.Children().FindDeep(content.IsOfType[*content.FootnoteRef]())).ToNot(BeNil(), "output: %q", output)
	})
})
