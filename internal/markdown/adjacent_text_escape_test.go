package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A program can make a paragraph from more than one text. When the texts
// are written next to each other, the joined text must not become markup,
// such as a page link or a tag, that no single text has.
var _ = Describe("Texts next to each other that join into markup", func() {
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

	textOf := func(root *content.Block) string {
		var b strings.Builder
		for _, n := range root.Children().FilterDeep(content.IsOfType[*content.Text]()) {
			b.WriteString(n.(*content.Text).Value)
		}

		return b.String()
	}

	It("does not make a page link from two texts", func() {
		// Given a paragraph of the texts `[[a` and `]]`
		paragraph := content.NewParagraph(content.NewText("[[a"), content.NewText("]]"))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then the paragraph has no page link and its text is `[[a]]`
		Expect(root.Children().FindDeep(content.IsOfType[*content.PageLink]())).To(BeNil(), "output: %q", output)
		Expect(textOf(root)).To(Equal("[[a]]"), "output: %q", output)
	})

	It("does not make a tag from two texts", func() {
		// Given a paragraph of the texts `x #` and `a`
		paragraph := content.NewParagraph(content.NewText("x #"), content.NewText("a"))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then the paragraph has no tag
		Expect(root.Children().FindDeep(content.IsOfType[*content.Hashtag]())).To(BeNil(), "output: %q", output)
	})

	It("escapes emphasis markers that are split over two texts", func() {
		// Given a paragraph of the texts `*a` and `b*`
		paragraph := content.NewParagraph(content.NewText("*a"), content.NewText("b*"))

		// When it is written, Then both markers are escaped
		Expect(write(paragraph)).To(Equal("\\*ab\\*"))
	})
})
