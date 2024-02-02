package content_test

import (
	"github.com/aholstenson/logseq-go/content"
	. "github.com/aholstenson/logseq-go/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Autoparagaph", func() {
	It("single text node", func() {
		nodes := content.AddAutomaticParagraphs([]content.Node{
			content.NewText("Hello world"),
		})

		Expect(nodes).To(EqualsNodes(content.NewParagraph(
			content.NewText("Hello world"),
		)))
	})

	It("multiple text nodes", func() {
		nodes := content.AddAutomaticParagraphs([]content.Node{
			content.NewText("Hello"),
			content.NewText("world"),
		})

		Expect(nodes).To(EqualsNodes(content.NewParagraph(
			content.NewText("Hello"),
			content.NewText("world"),
		)))
	})

	It("text node interrupted by block node", func() {
		nodes := content.AddAutomaticParagraphs([]content.Node{
			content.NewText("Hello"),
			content.NewHeading(1, content.NewText("world")),
		})

		Expect(nodes).To(EqualsNodes(
			content.NewParagraph(
				content.NewText("Hello"),
			),
			content.NewHeading(1, content.NewText("world")),
		))
	})

	It("text node interrupted by block node followed by text node", func() {
		nodes := content.AddAutomaticParagraphs([]content.Node{
			content.NewText("Hello"),
			content.NewHeading(1, content.NewText("world")),
			content.NewText("again"),
		})

		Expect(nodes).To(EqualsNodes(
			content.NewParagraph(
				content.NewText("Hello"),
			),
			content.NewHeading(1, content.NewText("world")),
			content.NewParagraph(
				content.NewText("again"),
			),
		))
	})

	It("text node interrupted by block node followed by text node interrupted by block node", func() {
		nodes := content.AddAutomaticParagraphs([]content.Node{
			content.NewText("Hello"),
			content.NewHeading(1, content.NewText("world")),
			content.NewText("again"),
			content.NewParagraph(content.NewText("and again")),
		})

		Expect(nodes).To(EqualsNodes(
			content.NewParagraph(
				content.NewText("Hello"),
			),
			content.NewHeading(1, content.NewText("world")),
			content.NewParagraph(
				content.NewText("again"),
			),
			content.NewParagraph(
				content.NewText("and again"),
			),
		))
	})
})
