package indexing_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/indexing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("References of blocks", func() {
	var idx *indexing.BlugeIndex

	BeforeEach(func() {
		idx = createIndex()

		indexPage(idx, "pages/a.md", "Page A",
			textBlock(text("parent"),
				textBlock([]content.Node{content.NewText("link "), content.NewPageLink("Target")}),
				textBlock([]content.Node{content.NewText("tag "), content.NewHashtag("tagged")}),
				textBlock([]content.Node{content.NewText("url "), content.NewAutoLink("https://example.com")}),
			),
		)
	})

	AfterEach(func() {
		Expect(idx.Close()).To(Succeed())
	})

	It("matches only the block with the reference", func() {
		Expect(previewsOf(searchBlocks(idx, indexing.References("Target")))).To(ConsistOf("link Target"))
	})

	It("matches only the block with the tag", func() {
		Expect(previewsOf(searchBlocks(idx, indexing.ReferencesTag("tagged")))).To(ConsistOf("tag #tagged"))
	})

	It("matches only the block with the link", func() {
		Expect(previewsOf(searchBlocks(idx, indexing.LinksToURL("https://example.com")))).To(ConsistOf("url"))
	})

	It("still matches the parent with PathReferences", func() {
		indexPage(idx, "pages/b.md", "Page B",
			textBlock([]content.Node{content.NewHashtag("tagged")},
				textBlock(text("child")),
			),
		)

		Expect(previewsOf(searchBlocks(idx, indexing.PathReferences("tagged")))).To(ConsistOf("tag #tagged", "#tagged", "child"))
	})

	It("matches the blocks directly below a block with an id", func() {
		withID := func(id string, value string, children ...*content.Block) *content.Block {
			block := textBlock(text(value), children...)
			block.Properties().Set("id", content.NewText(id))
			return block
		}

		indexPage(idx, "pages/b.md", "Page B",
			withID("parent-id", "parent",
				textBlock(text("first child"), textBlock(text("grandchild"))),
				withID("second-id", "second child"),
			),
			textBlock(text("no id"), textBlock(text("child of no id"))),
		)

		Expect(previewsOf(searchBlocks(idx, indexing.ParentIs("parent-id")))).To(ConsistOf("first child", "second child"))
		Expect(previewsOf(searchBlocks(idx, indexing.ParentIs("second-id")))).To(BeEmpty())

		Expect(previewsOf(searchBlocks(idx, indexing.AncestorIs("parent-id")))).To(ConsistOf("first child", "grandchild", "second child"))
		Expect(previewsOf(searchBlocks(idx, indexing.AncestorIs("second-id")))).To(BeEmpty())
	})

	It("does not add properties to blocks without an id", func() {
		block := textBlock(text("no id"), textBlock(text("child")))
		indexPage(idx, "pages/b.md", "Page B", block)

		Expect(block.FindProperties()).To(BeNil())
		Expect(block.Blocks()[0].FindProperties()).To(BeNil())
	})

	It("still matches the page with references in child blocks of its first block", func() {
		Expect(searchPages(idx, indexing.References("Target"))).To(HaveLen(1))
	})
})
