package indexing_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/indexing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// textBlock creates a block with a paragraph of the given nodes, followed by
// the given child blocks.
func textBlock(nodes []content.Node, children ...*content.Block) *content.Block {
	block := content.NewBlock(content.NewParagraph(nodes...))
	for _, child := range children {
		block.AddChild(child)
	}
	return block
}

func text(value string) []content.Node {
	return []content.Node{content.NewText(value)}
}

var _ = Describe("PathReferences", func() {
	var idx *indexing.BlugeIndex

	BeforeEach(func() {
		idx = createIndex()
	})

	AfterEach(func() {
		Expect(idx.Close()).To(Succeed())
	})

	It("matches blocks that reference the page", func() {
		indexPage(idx, "pages/a.md", "Page A",
			textBlock([]content.Node{content.NewText("buy "), content.NewPageLink("Shopping")}),
			textBlock(text("other")),
		)

		results := searchBlocks(idx, indexing.PathReferences("Shopping"))
		Expect(previewsOf(results)).To(ConsistOf("buy Shopping"))
	})

	It("matches blocks that reference the page with a tag", func() {
		indexPage(idx, "pages/a.md", "Page A",
			textBlock([]content.Node{content.NewText("milk "), content.NewHashtag("shopping")}),
		)

		results := searchBlocks(idx, indexing.PathReferences("shopping"))
		Expect(previewsOf(results)).To(ConsistOf("milk #shopping"))
	})

	It("matches the blocks below a block that references the page", func() {
		indexPage(idx, "pages/a.md", "Page A",
			textBlock([]content.Node{content.NewHashtag("shopping")},
				textBlock(text("milk"),
					textBlock(text("oat")),
				),
				textBlock(text("bread")),
			),
			textBlock(text("sibling")),
		)

		results := searchBlocks(idx, indexing.PathReferences("shopping"))
		Expect(previewsOf(results)).To(ConsistOf("#shopping", "milk", "oat", "bread"))
	})

	It("does not match the parents of a block that references the page", func() {
		indexPage(idx, "pages/a.md", "Page A",
			textBlock(text("parent"),
				textBlock([]content.Node{content.NewHashtag("shopping")}),
			),
		)

		results := searchBlocks(idx, indexing.PathReferences("shopping"))
		Expect(previewsOf(results)).To(ConsistOf("#shopping"))
	})

	It("matches all blocks on the page", func() {
		indexPage(idx, "pages/shopping.md", "Shopping",
			textBlock(text("milk"),
				textBlock(text("oat")),
			),
		)
		indexPage(idx, "pages/other.md", "Other",
			textBlock(text("unrelated")),
		)

		results := searchBlocks(idx, indexing.PathReferences("shopping"))
		Expect(previewsOf(results)).To(ConsistOf("milk", "oat"))
	})

	It("matches references that differ in case", func() {
		indexPage(idx, "pages/a.md", "Page A",
			textBlock([]content.Node{content.NewPageLink("reading list")},
				textBlock(text("a book")),
			),
		)

		results := searchBlocks(idx, indexing.PathReferences("Reading List"))
		Expect(previewsOf(results)).To(ConsistOf("reading list", "a book"))
	})

	It("does not match pages", func() {
		indexPage(idx, "pages/a.md", "Page A",
			textBlock([]content.Node{content.NewPageLink("Shopping")}),
		)

		Expect(searchPages(idx, indexing.PathReferences("Shopping"))).To(BeEmpty())
	})

	It("leaves out a part of the graph when used with Not", func() {
		indexPage(idx, "pages/a.md", "Page A",
			textBlock([]content.Node{content.NewHashtag("shopping")},
				taskBlock(content.TaskStatusTodo, content.PriorityNone, "buy milk"),
			),
			taskBlock(content.TaskStatusTodo, content.PriorityNone, "water plants"),
		)

		results := searchBlocks(idx, indexing.And(
			indexing.IsOpenTask(),
			indexing.Not(indexing.PathReferences("shopping")),
		))
		Expect(previewsOf(results)).To(ConsistOf("water plants"))
	})

	It("stops matching when the reference is removed", func() {
		indexPage(idx, "pages/a.md", "Page A",
			textBlock([]content.Node{content.NewHashtag("shopping")},
				textBlock(text("milk")),
			),
		)
		indexPage(idx, "pages/a.md", "Page A",
			textBlock(text("list"),
				textBlock(text("milk")),
			),
		)

		Expect(searchBlocks(idx, indexing.PathReferences("shopping"))).To(BeEmpty())
	})
})
