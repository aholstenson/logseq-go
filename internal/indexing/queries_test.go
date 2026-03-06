package indexing_test

import (
	"context"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/indexing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func createIndex() *indexing.BlugeIndex {
	idx, err := indexing.NewBlugeIndex(nil, "")
	Expect(err).ToNot(HaveOccurred())
	return idx
}

func indexPage(idx *indexing.BlugeIndex, subPath string, title string, blocks ...*content.Block) {
	ctx := context.Background()
	page := &indexing.Page{
		SubPath:      subPath,
		Type:         indexing.PageTypeDedicated,
		LastModified: time.Now(),
		Title:        title,
		Blocks:       blocks,
	}
	Expect(idx.IndexPage(ctx, page)).To(Succeed())
	Expect(idx.Sync()).To(Succeed())
}

func searchPages(idx *indexing.BlugeIndex, query indexing.Query) []*indexing.Page {
	ctx := context.Background()
	results, err := idx.SearchPages(ctx, query, indexing.SearchOptions{})
	Expect(err).ToNot(HaveOccurred())
	return results.Results()
}

func searchBlocks(idx *indexing.BlugeIndex, query indexing.Query) []*indexing.Block {
	ctx := context.Background()
	results, err := idx.SearchBlocks(ctx, query, indexing.SearchOptions{})
	Expect(err).ToNot(HaveOccurred())
	return results.Results()
}

var _ = Describe("Queries", func() {
	var idx *indexing.BlugeIndex

	BeforeEach(func() {
		idx = createIndex()
	})

	AfterEach(func() {
		Expect(idx.Close()).To(Succeed())
	})

	Describe("All", func() {
		It("matches all pages", func() {
			indexPage(idx, "pages/a.md", "Alpha",
				content.NewBlock(content.NewParagraph(content.NewText("hello"))),
			)
			indexPage(idx, "pages/b.md", "Beta",
				content.NewBlock(content.NewParagraph(content.NewText("world"))),
			)

			results := searchPages(idx, indexing.All())
			Expect(results).To(HaveLen(2))
		})
	})

	Describe("None", func() {
		It("matches no pages", func() {
			indexPage(idx, "pages/a.md", "Alpha",
				content.NewBlock(content.NewParagraph(content.NewText("hello"))),
			)

			results := searchPages(idx, indexing.None())
			Expect(results).To(BeEmpty())
		})
	})

	Describe("And", func() {
		It("requires all clauses to match", func() {
			indexPage(idx, "pages/a.md", "Alpha",
				content.NewBlock(content.NewParagraph(content.NewText("hello world"))),
			)
			indexPage(idx, "pages/b.md", "Beta",
				content.NewBlock(content.NewParagraph(content.NewText("hello"))),
			)

			results := searchPages(idx, indexing.And(
				indexing.TitleMatches("Alpha"),
				indexing.ContentMatches("hello"),
			))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Alpha"))
		})
	})

	Describe("Or", func() {
		It("matches if any clause matches", func() {
			indexPage(idx, "pages/a.md", "Alpha",
				content.NewBlock(content.NewParagraph(content.NewText("hello"))),
			)
			indexPage(idx, "pages/b.md", "Beta",
				content.NewBlock(content.NewParagraph(content.NewText("world"))),
			)
			indexPage(idx, "pages/c.md", "Gamma",
				content.NewBlock(content.NewParagraph(content.NewText("nothing"))),
			)

			results := searchPages(idx, indexing.Or(
				indexing.TitleMatches("Alpha"),
				indexing.TitleMatches("Beta"),
			))
			Expect(results).To(HaveLen(2))
		})
	})

	Describe("Not", func() {
		It("excludes matching pages", func() {
			indexPage(idx, "pages/a.md", "Alpha",
				content.NewBlock(content.NewParagraph(content.NewText("hello"))),
			)
			indexPage(idx, "pages/b.md", "Beta",
				content.NewBlock(content.NewParagraph(content.NewText("world"))),
			)

			results := searchPages(idx, indexing.And(
				indexing.All(),
				indexing.Not(indexing.TitleMatches("Alpha")),
			))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Beta"))
		})
	})

	Describe("TitleMatches", func() {
		It("matches pages by title", func() {
			indexPage(idx, "pages/a.md", "Hello World",
				content.NewBlock(content.NewParagraph(content.NewText("content"))),
			)
			indexPage(idx, "pages/b.md", "Goodbye",
				content.NewBlock(content.NewParagraph(content.NewText("content"))),
			)

			results := searchPages(idx, indexing.TitleMatches("Hello World"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Hello World"))
		})

		It("requires all words to match", func() {
			indexPage(idx, "pages/a.md", "Hello World",
				content.NewBlock(content.NewParagraph(content.NewText("content"))),
			)
			indexPage(idx, "pages/b.md", "Hello There",
				content.NewBlock(content.NewParagraph(content.NewText("content"))),
			)

			results := searchPages(idx, indexing.TitleMatches("Hello World"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Hello World"))
		})
	})

	Describe("TitlePartiallyMatches", func() {
		It("matches pages where any word matches", func() {
			indexPage(idx, "pages/a.md", "Hello World",
				content.NewBlock(content.NewParagraph(content.NewText("content"))),
			)
			indexPage(idx, "pages/b.md", "Hello There",
				content.NewBlock(content.NewParagraph(content.NewText("content"))),
			)
			indexPage(idx, "pages/c.md", "Goodbye",
				content.NewBlock(content.NewParagraph(content.NewText("content"))),
			)

			results := searchPages(idx, indexing.TitlePartiallyMatches("Hello World"))
			Expect(results).To(HaveLen(2))
		})

		It("matches a single word", func() {
			indexPage(idx, "pages/a.md", "Hello World",
				content.NewBlock(content.NewParagraph(content.NewText("content"))),
			)
			indexPage(idx, "pages/b.md", "Goodbye",
				content.NewBlock(content.NewParagraph(content.NewText("content"))),
			)

			results := searchPages(idx, indexing.TitlePartiallyMatches("Hello"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Hello World"))
		})
	})

	Describe("ContentMatches", func() {
		It("matches pages by content", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(content.NewParagraph(content.NewText("the quick brown fox"))),
			)
			indexPage(idx, "pages/b.md", "Page B",
				content.NewBlock(content.NewParagraph(content.NewText("the lazy dog"))),
			)

			results := searchPages(idx, indexing.ContentMatches("quick brown fox"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Page A"))
		})

		It("matches blocks by content", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(content.NewParagraph(content.NewText("first block"))),
				content.NewBlock(content.NewParagraph(content.NewText("second block with keywords"))),
			)

			results := searchBlocks(idx, indexing.ContentMatches("keywords"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].PageSubPath).To(Equal("pages/a.md"))
		})
	})

	Describe("PropertyMatches", func() {
		It("matches pages by property text", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(
					content.NewProperties(
						content.NewProperty("category", content.NewText("science fiction")),
					),
					content.NewParagraph(content.NewText("content")),
				),
			)
			indexPage(idx, "pages/b.md", "Page B",
				content.NewBlock(
					content.NewProperties(
						content.NewProperty("category", content.NewText("history")),
					),
					content.NewParagraph(content.NewText("content")),
				),
			)

			results := searchPages(idx, indexing.PropertyMatches("category", "science"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Page A"))
		})
	})

	Describe("PropertyEquals", func() {
		It("matches pages by exact property value", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(
					content.NewProperties(
						content.NewProperty("status", content.NewText("done")),
					),
					content.NewParagraph(content.NewText("content")),
				),
			)
			indexPage(idx, "pages/b.md", "Page B",
				content.NewBlock(
					content.NewProperties(
						content.NewProperty("status", content.NewText("in progress")),
					),
					content.NewParagraph(content.NewText("content")),
				),
			)

			results := searchPages(idx, indexing.PropertyEquals("status", "done"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Page A"))
		})
	})

	Describe("References", func() {
		It("matches pages that reference another page", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(
					content.NewParagraph(
						content.NewText("see "),
						content.NewPageLink("Target"),
					),
				),
			)
			indexPage(idx, "pages/b.md", "Page B",
				content.NewBlock(content.NewParagraph(content.NewText("no refs"))),
			)

			results := searchPages(idx, indexing.References("Target"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Page A"))
		})
	})

	Describe("ReferencesTag", func() {
		It("matches pages that reference a tag", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(
					content.NewParagraph(
						content.NewText("tagged "),
						content.NewHashtag("important"),
					),
				),
			)
			indexPage(idx, "pages/b.md", "Page B",
				content.NewBlock(content.NewParagraph(content.NewText("no tags"))),
			)

			results := searchPages(idx, indexing.ReferencesTag("important"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Page A"))
		})
	})

	Describe("LinksToURL", func() {
		It("matches pages that link to a URL", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(
					content.NewParagraph(
						content.NewText("check "),
						content.NewAutoLink("https://example.com"),
					),
				),
			)
			indexPage(idx, "pages/b.md", "Page B",
				content.NewBlock(content.NewParagraph(content.NewText("no links"))),
			)

			results := searchPages(idx, indexing.LinksToURL("https://example.com"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Page A"))
		})
	})
})
