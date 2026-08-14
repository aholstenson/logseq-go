package content_test

import (
	"github.com/aholstenson/logseq-go/content"
	. "github.com/aholstenson/logseq-go/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Querying", func() {
	Describe("Page references", func() {
		It("finds the references in a block", func() {
			block := content.NewBlock(
				content.NewParagraph(
					content.NewText("see "),
					content.NewPageLink("Linked"),
					content.NewText(" and "),
					content.NewHashtag("Tagged"),
				),
			)

			Expect(block.Children().PageReferences()).To(EqualsNodes(
				content.NewPageLink("Linked"),
				content.NewHashtag("Tagged"),
			))
		})

		It("finds the references in the value of a property", func() {
			block := content.NewBlock(
				content.NewProperties(
					content.NewProperty("tags", content.NewPageRefText("Referenced")),
				),
			)

			Expect(block.Children().PageReferences()).To(EqualsNodes(
				content.NewPageRefText("Referenced"),
			))
		})

		It("leaves out the value of a property whose references are ignored", func() {
			block := content.NewBlock(
				content.NewProperties(
					content.NewProperty("author", content.NewPageLink("Someone")).
						WithPageRefsIgnored(true),
				),
			)

			Expect(block.Children().PageReferences()).To(BeEmpty())
		})
	})
})
