package logseq_test

import (
	"context"
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aliases", func() {
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
			graph = nil
		}
	})

	aliasesOf := func(title string) []string {
		page, err := graph.OpenPage(title)
		Expect(err).ToNot(HaveOccurred())
		return page.Aliases()
	}

	Describe("Aliases of a page", func() {
		It("reads an alias written as text", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: Other\n- content of target\n",
			})

			Expect(aliasesOf("target")).To(Equal([]string{"Other"}))
		})

		It("reads an alias written as a page reference", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: [[Other]]\n- content of target\n",
			})

			Expect(aliasesOf("target")).To(Equal([]string{"Other"}))
		})

		It("reads several aliases separated by commas", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: Other, Second Name\n- content of target\n",
			})

			Expect(aliasesOf("target")).To(Equal([]string{"Other", "Second Name"}))
		})

		It("reads several aliases written as page references", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: [[Other]], [[Second Name]]\n- content of target\n",
			})

			Expect(aliasesOf("target")).To(Equal([]string{"Other", "Second Name"}))
		})

		It("reads aliases that mix references and text", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: [[Other]], Second Name\n- content of target\n",
			})

			Expect(aliasesOf("target")).To(Equal([]string{"Other", "Second Name"}))
		})

		It("has no aliases without an alias property", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "- content of target\n",
			})

			Expect(aliasesOf("target")).To(BeEmpty())
		})
	})

	Describe("Opening a page by an alias", func() {
		It("opens the page the alias belongs to", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: Other\n- content of target\n",
			})

			page, err := graph.OpenPage("Other")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.IsNew()).To(BeFalse())
			Expect(page.Title()).To(Equal("target"))
			Expect(page.Blocks()).To(HaveLen(2))
		})

		It("opens the page for an alias written as a page reference", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: [[Other]]\n- content of target\n",
			})

			page, err := graph.OpenPage("Other")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.Title()).To(Equal("target"))
		})

		It("opens the page for any of several aliases", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: Other, Second Name\n- content of target\n",
			})

			first, err := graph.OpenPage("Other")
			Expect(err).ToNot(HaveOccurred())
			Expect(first.Title()).To(Equal("target"))

			second, err := graph.OpenPage("Second Name")
			Expect(err).ToNot(HaveOccurred())
			Expect(second.Title()).To(Equal("target"))
		})

		It("opens the page for an alias that differs in case", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: Other\n- content of target\n",
			})

			page, err := graph.OpenPage("OTHER")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.Title()).To(Equal("target"))
		})

		It("prefers a page of its own over an alias of another page", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: Other\n- content of target\n",
				"Other.md":  "- content of other\n",
			})

			page, err := graph.OpenPage("Other")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.Title()).To(Equal("Other"))
		})

		It("creates a new page for a title that is not an alias", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: Other\n- content of target\n",
			})

			page, err := graph.OpenPage("Unknown")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.IsNew()).To(BeTrue())
			Expect(page.Title()).To(Equal("Unknown"))
		})

		It("creates a new page when indexing is not enabled", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "target.md"),
				[]byte("alias:: Other\n- content of target\n"),
				0o644,
			)).To(Succeed())

			var err error
			graph, err = logseq.Open(ctx, dir)
			Expect(err).ToNot(HaveOccurred())

			// Without an index there is nothing to look aliases up in
			page, err := graph.OpenPage("Other")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.IsNew()).To(BeTrue())
		})

		It("finds the page by its alias with a query", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: Other\n- content of target\n",
			})

			results, err := graph.SearchPages(ctx, logseq.WithQuery(logseq.HasAlias("Other")))
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))
			Expect(results.Results()[0].Title()).To(Equal("target"))
		})
	})

	Describe("Opening a page by an alias in a transaction", func() {
		It("gives the same page as opening it by its title", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: Other\n- content of target\n",
			})

			tx := graph.NewTransaction()
			byAlias, err := tx.OpenPage("Other")
			Expect(err).ToNot(HaveOccurred())

			byTitle, err := tx.OpenPage("target")
			Expect(err).ToNot(HaveOccurred())

			Expect(byAlias).To(BeIdenticalTo(byTitle))
		})

		It("gives the same page when opened by its title first", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: Other\n- content of target\n",
			})

			tx := graph.NewTransaction()
			byTitle, err := tx.OpenPage("target")
			Expect(err).ToNot(HaveOccurred())

			byAlias, err := tx.OpenPage("Other")
			Expect(err).ToNot(HaveOccurred())

			Expect(byAlias).To(BeIdenticalTo(byTitle))
		})

		It("saves changes to the file of the page the alias belongs to", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: Other\n- content of target\n",
			})

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("Other")
			Expect(err).ToNot(HaveOccurred())

			page.AddBlock(textBlock("added via the alias"))
			Expect(tx.Save()).To(Succeed())

			data, err := os.ReadFile(filepath.Join(dir, "pages", "target.md"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal(
				"alias:: Other\n- content of target\n- added via the alias\n",
			))

			Expect(filepath.Join(dir, "pages", "Other.md")).ToNot(BeAnExistingFile())
		})
	})
})
