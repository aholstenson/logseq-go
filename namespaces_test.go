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

var _ = Describe("Namespaces", func() {
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

	// titles returns the title of every page in a result set.
	titles := func(results logseq.SearchResults[logseq.PageResult]) []string {
		titles := make([]string, 0, results.Size())
		for _, result := range results.Results() {
			titles = append(titles, result.Title())
		}
		return titles
	}

	children := func(title string, opts ...logseq.SearchOption) logseq.SearchResults[logseq.PageResult] {
		page, err := graph.OpenPage(title)
		Expect(err).ToNot(HaveOccurred())

		results, err := page.NamespaceChildren(ctx, opts...)
		Expect(err).ToNot(HaveOccurred())
		return results
	}

	Describe("Namespace of a page", func() {
		It("is empty for a page that is not in a namespace", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"page.md": "- content\n",
			})

			page, err := graph.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.Namespace()).To(Equal(""))
		})

		It("is the part of the title before the last slash", func() {
			// Logseq stores the / of a namespace as ___ in the file name
			graph = openGraphWithPages(dir, map[string]string{
				"parent___child.md": "- content\n",
			})

			page, err := graph.OpenPage("parent/child")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.IsNew()).To(BeFalse())
			Expect(page.Namespace()).To(Equal("parent"))
		})

		It("keeps the parts of a nested namespace", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"parent___child___grandchild.md": "- content\n",
			})

			page, err := graph.OpenPage("parent/child/grandchild")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.Namespace()).To(Equal("parent/child"))
		})

		It("is empty for a journal", func() {
			var err error
			graph, err = logseq.Open(ctx, dir)
			Expect(err).ToNot(HaveOccurred())

			journal, err := graph.OpenJournal(time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local))
			Expect(err).ToNot(HaveOccurred())
			Expect(journal.Namespace()).To(Equal(""))
		})
	})

	Describe("Children of a namespace", func() {
		It("finds the pages directly in the namespace", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"parent.md":                "- content of parent\n",
				"parent___child.md":        "- content of child\n",
				"parent___other.md":        "- content of other\n",
				"unrelated.md":             "- content of unrelated\n",
				"unrelated___stepchild.md": "- content of stepchild\n",
			})

			Expect(titles(children("parent"))).To(ConsistOf("parent/child", "parent/other"))
		})

		It("leaves out the pages deeper in the namespace", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"parent.md":                      "- content of parent\n",
				"parent___child.md":              "- content of child\n",
				"parent___child___grandchild.md": "- content of grandchild\n",
			})

			Expect(titles(children("parent"))).To(ConsistOf("parent/child"))
			Expect(titles(children("parent/child"))).To(ConsistOf("parent/child/grandchild"))
		})

		It("finds children of a page that does not exist itself", func() {
			// Logseq allows a namespace without a page of its own
			graph = openGraphWithPages(dir, map[string]string{
				"parent___child.md": "- content of child\n",
			})

			page, err := graph.OpenPage("parent")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.IsNew()).To(BeTrue())

			results, err := page.NamespaceChildren(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(titles(results)).To(ConsistOf("parent/child"))
		})

		It("finds children of a namespace that differs in case", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"Parent___child.md": "- content of child\n",
			})

			Expect(titles(children("parent"))).To(ConsistOf("Parent/child"))
		})

		It("finds nothing for a page without children", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"parent.md": "- content of parent\n",
			})

			Expect(children("parent").Size()).To(BeZero())
		})

		It("reports the total number of children while returning a page of them", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"parent___a.md": "- a\n",
				"parent___b.md": "- b\n",
				"parent___c.md": "- c\n",
			})

			results := children("parent", logseq.WithMaxHits(2))
			Expect(results.Size()).To(Equal(2))
			Expect(results.Count()).To(Equal(3))
		})

		It("opens a child that was found", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"parent___child.md": "- content of child\n",
			})

			results := children("parent")
			Expect(results.Size()).To(Equal(1))

			child, err := results.Results()[0].Open()
			Expect(err).ToNot(HaveOccurred())
			Expect(child.Title()).To(Equal("parent/child"))
			Expect(child.Namespace()).To(Equal("parent"))
		})

		It("saves changes to a child opened in a transaction", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"parent___child.md": "- content of child\n",
			})

			tx := graph.NewTransaction()
			parent, err := tx.OpenPage("parent")
			Expect(err).ToNot(HaveOccurred())

			results, err := parent.NamespaceChildren(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))

			child, err := results.Results()[0].Open()
			Expect(err).ToNot(HaveOccurred())
			child.AddBlock(textBlock("added to the child"))

			Expect(tx.Save()).To(Succeed())

			data, err := os.ReadFile(filepath.Join(dir, "pages", "parent___child.md"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("- content of child\n- added to the child\n"))
		})

		It("fails when indexing is not enabled", func() {
			var err error
			graph, err = logseq.Open(ctx, dir)
			Expect(err).ToNot(HaveOccurred())

			page, err := graph.OpenPage("parent")
			Expect(err).ToNot(HaveOccurred())

			_, err = page.NamespaceChildren(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Namespace queries", func() {
		It("finds the pages directly in a namespace", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"parent___child.md":              "- content of child\n",
				"parent___child___grandchild.md": "- content of grandchild\n",
			})

			results, err := graph.SearchPages(ctx, logseq.WithQuery(logseq.InNamespace("parent")))
			Expect(err).ToNot(HaveOccurred())
			Expect(titles(results)).To(ConsistOf("parent/child"))
		})

		It("finds every page under a namespace", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"parent.md":                      "- content of parent\n",
				"parent___child.md":              "- content of child\n",
				"parent___child___grandchild.md": "- content of grandchild\n",
			})

			results, err := graph.SearchPages(ctx, logseq.WithQuery(logseq.UnderNamespace("parent")))
			Expect(err).ToNot(HaveOccurred())
			Expect(titles(results)).To(ConsistOf("parent/child", "parent/child/grandchild"))
		})
	})
})
