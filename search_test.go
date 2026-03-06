package logseq_test

import (
	"context"
	"os"
	"path/filepath"
	"time"

	logseq "github.com/aholstenson/logseq-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func openGraphWithPages(dir string, pages map[string]string) *logseq.Graph {
	for name, content := range pages {
		Expect(os.WriteFile(
			filepath.Join(dir, "pages", name),
			[]byte(content),
			0o644,
		)).To(Succeed())
	}

	graph, err := logseq.Open(context.Background(), dir, logseq.WithInMemoryIndex())
	Expect(err).ToNot(HaveOccurred())
	return graph
}

var _ = Describe("Search", func() {
	var (
		graph *logseq.Graph
		dir   string
		ctx   context.Context
	)

	BeforeEach(func() {
		dir = setupGraph()
		ctx = context.Background()
	})

	AfterEach(func() {
		if graph != nil {
			graph.Close()
		}
	})

	Describe("SearchPages", func() {
		It("finds pages by title", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"alpha.md": "- content of alpha\n",
				"beta.md":  "- content of beta\n",
			})

			results, err := graph.SearchPages(ctx,
				logseq.WithQuery(logseq.TitleMatches("alpha")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))
			Expect(results.Results()[0].Title()).To(Equal("alpha"))
		})

		It("finds pages by content", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"alpha.md": "- the quick brown fox\n",
				"beta.md":  "- the lazy dog\n",
			})

			results, err := graph.SearchPages(ctx,
				logseq.WithQuery(logseq.ContentMatches("quick brown fox")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))
			Expect(results.Results()[0].Title()).To(Equal("alpha"))
		})

		It("can open a found page", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"alpha.md": "- hello world\n",
			})

			results, err := graph.SearchPages(ctx,
				logseq.WithQuery(logseq.TitleMatches("alpha")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))

			page, err := results.Results()[0].Open()
			Expect(err).ToNot(HaveOccurred())
			Expect(page).ToNot(BeNil())
			Expect(page.Title()).To(Equal("alpha"))
			Expect(page.Blocks()).To(HaveLen(1))
		})

		It("reports correct result type for dedicated pages", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"alpha.md": "- content\n",
			})

			results, err := graph.SearchPages(ctx,
				logseq.WithQuery(logseq.TitleMatches("alpha")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Results()[0].Type()).To(Equal(logseq.PageTypeDedicated))
		})

		It("finds journal pages", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "journals", "2025_06_15.md"),
				[]byte("- journal entry xyzzy\n"),
				0o644,
			)).To(Succeed())

			graph = openGraphWithPages(dir, map[string]string{})

			results, err := graph.SearchPages(ctx,
				logseq.WithQuery(logseq.ContentMatches("xyzzy")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))
			Expect(results.Results()[0].Type()).To(Equal(logseq.PageTypeJournal))
			Expect(results.Results()[0].Date()).To(Equal(
				time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			))
		})
	})

	Describe("SearchBlocks", func() {
		It("can open the last block on a page via location", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"multiblock.md": "- first block\n- second block\n- last block with unique7content\n",
			})

			results, err := graph.SearchBlocks(ctx,
				logseq.WithQuery(logseq.ContentMatches("unique7content")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))

			block, page, err := results.Results()[0].Open()
			Expect(err).ToNot(HaveOccurred())
			Expect(block).ToNot(BeNil())
			Expect(page).ToNot(BeNil())
		})

		It("can open a block that is not the last one", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"multiblock.md": "- target8block here\n- another block\n- final block\n",
			})

			results, err := graph.SearchBlocks(ctx,
				logseq.WithQuery(logseq.ContentMatches("target8block")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))

			block, page, err := results.Results()[0].Open()
			Expect(err).ToNot(HaveOccurred())
			Expect(block).ToNot(BeNil())
			Expect(page).ToNot(BeNil())
		})

		It("can open a nested child block", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"nested.md": "- parent block\n\t- nested9child here\n",
			})

			results, err := graph.SearchBlocks(ctx,
				logseq.WithQuery(logseq.ContentMatches("nested9child")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))

			block, page, err := results.Results()[0].Open()
			Expect(err).ToNot(HaveOccurred())
			Expect(block).ToNot(BeNil())
			Expect(page).ToNot(BeNil())
		})

		It("reports correct page metadata on block results", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"mypage.md": "- some unique3content here\n",
			})

			results, err := graph.SearchBlocks(ctx,
				logseq.WithQuery(logseq.ContentMatches("unique3content")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))

			result := results.Results()[0]
			Expect(result.PageTitle()).To(Equal("mypage"))
			Expect(result.PageType()).To(Equal(logseq.PageTypeDedicated))
			Expect(result.Preview()).ToNot(BeEmpty())
		})

		It("reports journal metadata on block results", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "journals", "2025_03_20.md"),
				[]byte("- journal unique4entry\n"),
				0o644,
			)).To(Succeed())

			graph = openGraphWithPages(dir, map[string]string{})

			results, err := graph.SearchBlocks(ctx,
				logseq.WithQuery(logseq.ContentMatches("unique4entry")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))

			result := results.Results()[0]
			Expect(result.PageType()).To(Equal(logseq.PageTypeJournal))
			Expect(result.PageDate()).To(Equal(
				time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC),
			))
		})

		It("can open a block with a stable ID", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"withid.md": "- id:: 65a1b2c3-d4e5-6789-abcd-ef0123456789\n  some block\n- id:: aaaa1111-bb22-cc33-dd44-eeeeeeee5555\n  unique5stable content\n",
			})

			results, err := graph.SearchBlocks(ctx,
				logseq.WithQuery(logseq.ContentMatches("unique5stable")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))
			Expect(results.Results()[0].ID()).To(Equal("aaaa1111-bb22-cc33-dd44-eeeeeeee5555"))

			block, page, err := results.Results()[0].Open()
			Expect(err).ToNot(HaveOccurred())
			Expect(block).ToNot(BeNil())
			Expect(page).ToNot(BeNil())
			Expect(block.ID()).To(Equal("aaaa1111-bb22-cc33-dd44-eeeeeeee5555"))
		})
	})

	Describe("Search options", func() {
		It("limits results with WithMaxHits", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"a.md": "- common searchterm99\n",
				"b.md": "- common searchterm99\n",
				"c.md": "- common searchterm99\n",
			})

			results, err := graph.SearchPages(ctx,
				logseq.WithQuery(logseq.ContentMatches("searchterm99")),
				logseq.WithMaxHits(2),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(2))
			Expect(results.Count()).To(Equal(3))
		})

		It("paginates with FromHit", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"a.md": "- common paginateterm1\n",
				"b.md": "- common paginateterm1\n",
				"c.md": "- common paginateterm1\n",
			})

			results, err := graph.SearchPages(ctx,
				logseq.WithQuery(logseq.ContentMatches("paginateterm1")),
				logseq.WithMaxHits(2),
				logseq.FromHit(2),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))
			Expect(results.Count()).To(Equal(3))
		})
	})
})
