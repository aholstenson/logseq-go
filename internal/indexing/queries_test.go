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
	if len(blocks) > 0 {
		if properties, ok := blocks[0].FirstChild().(*content.Properties); ok {
			page.Properties = properties
		}
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

		It("matches pages by raw text content", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(content.NewParagraph(
					content.NewRawText("**pre-generated** content"),
				)),
			)

			results := searchPages(idx, indexing.ContentMatches("pre-generated"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Page A"))
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

	Describe("HasProperty", func() {
		It("matches pages that have the property", func() {
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
						content.NewProperty("other", content.NewText("value")),
					),
					content.NewParagraph(content.NewText("content")),
				),
			)

			results := searchPages(idx, indexing.HasProperty("status"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Page A"))
		})

		It("matches blocks that have the property", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(content.NewParagraph(content.NewText("first"))),
				content.NewBlock(
					content.NewParagraph(content.NewText("Daily")),
					content.NewProperties(
						content.NewProperty("template", content.NewText("Daily")),
					),
				),
			)

			results := searchBlocks(idx, indexing.HasProperty("template"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Preview).To(Equal("Daily"))
		})

		It("matches a property with an empty value", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(
					content.NewProperties(content.NewProperty("tags")),
					content.NewParagraph(content.NewText("content")),
				),
			)

			results := searchPages(idx, indexing.HasProperty("tags"))
			Expect(results).To(HaveLen(1))
		})

		It("matches the property name without regard for case", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(
					content.NewProperties(
						content.NewProperty("Status", content.NewText("done")),
					),
					content.NewParagraph(content.NewText("content")),
				),
			)

			Expect(searchPages(idx, indexing.HasProperty("status"))).To(HaveLen(1))
			Expect(searchPages(idx, indexing.HasProperty("STATUS"))).To(HaveLen(1))
		})
	})

	Describe("Page properties", func() {
		It("returns the property values of found pages", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(
					content.NewProperties(
						content.NewProperty("icon", content.NewText("📚")),
						content.NewProperty("page-type", content.NewPageLink("book")),
					),
					content.NewParagraph(content.NewText("content")),
				),
			)

			results := searchPages(idx, indexing.PropertyReferences("page-type", "Book"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].PropertyValues).To(Equal(map[string]string{
				"icon":      "📚",
				"page-type": "book",
			}))
		})

		It("returns no property values for pages without properties", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(content.NewParagraph(content.NewText("content"))),
			)

			results := searchPages(idx, indexing.TitleMatches("Page A"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].PropertyValues).To(BeNil())
		})
	})

	Describe("Block properties", func() {
		It("returns the property values of found blocks", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(content.NewParagraph(content.NewText("first"))),
				content.NewBlock(
					content.NewParagraph(content.NewText("Daily")),
					content.NewProperties(
						content.NewProperty("template", content.NewText("Daily")),
						content.NewProperty("type", content.NewPageLink("Journal")),
					),
				),
			)

			results := searchBlocks(idx, indexing.PropertyEquals("template", "Daily"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].PropertyValues).To(Equal(map[string]string{
				"template": "Daily",
				"type":     "Journal",
			}))
		})

		It("returns no property values for blocks without properties", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(content.NewParagraph(content.NewText("first"))),
			)

			results := searchBlocks(idx, indexing.ContentMatches("first"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].PropertyValues).To(BeNil())
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

		It("matches references that differ in case", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(
					content.NewParagraph(
						content.NewText("see "),
						content.NewPageLink("target"),
					),
				),
			)

			results := searchPages(idx, indexing.References("Target"))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Page A"))
		})

		It("matches pages that embed another page", func() {
			indexPage(idx, "pages/a.md", "Page A",
				content.NewBlock(
					content.NewParagraph(
						content.NewPageEmbed("Target"),
					),
				),
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
