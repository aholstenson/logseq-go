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

	Describe("Queries", func() {
		simple := content.NewQuery("(task TODO)")
		advanced := content.NewQueryCommand("{:query [:find ?b :where [?b :block/marker \"TODO\"]]}")
		inner := content.NewBlock(content.NewParagraph(content.NewText("see "), simple))
		outer := content.NewBlock(advanced, inner)

		It("finds simple and advanced queries", func() {
			Expect(outer.Children().FilterDeep(content.IsQuery())).To(EqualsNodes(advanced, simple))
		})

		It("finds the block that holds a node", func() {
			Expect(content.BlockOf(simple)).To(BeIdenticalTo(inner))
			Expect(content.BlockOf(advanced)).To(BeIdenticalTo(outer))
			Expect(content.BlockOf(inner)).To(BeIdenticalTo(inner))
		})

		It("returns nil for a node that is not in a block", func() {
			Expect(content.BlockOf(content.NewText("alone"))).To(BeNil())
			Expect(content.BlockOf(nil)).To(BeNil())
		})
	})
})
