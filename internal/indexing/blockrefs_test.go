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

	It("still matches the page with references in child blocks of its first block", func() {
		Expect(searchPages(idx, indexing.References("Target"))).To(HaveLen(1))
	})
})
