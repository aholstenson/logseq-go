package logseq_test

import (
	"context"
	"os"
	"path/filepath"
	"time"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Linked references", func() {
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

	// previews returns the preview of every reference, which is the text of the
	// block that references the page.
	previews := func(results logseq.SearchResults[logseq.BlockResult]) []string {
		previews := make([]string, 0, results.Size())
		for _, result := range results.Results() {
			previews = append(previews, result.Preview())
		}
		return previews
	}

	linkedReferences := func(title string, opts ...logseq.SearchOption) logseq.SearchResults[logseq.BlockResult] {
		page, err := graph.OpenPage(title)
		Expect(err).ToNot(HaveOccurred())

		results, err := page.LinkedReferences(ctx, opts...)
		Expect(err).ToNot(HaveOccurred())
		return results
	}

	It("finds blocks that link to the page", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md":    "- content of target\n",
			"referrer.md":  "- see [[target]] for more\n",
			"unrelated.md": "- nothing to see here\n",
		})

		results := linkedReferences("target")
		Expect(results.Size()).To(Equal(1))
		Expect(previews(results)).To(ConsistOf("see target for more"))
	})

	It("finds blocks that tag the page", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md":   "- content of target\n",
			"referrer.md": "- tagged #target here\n",
		})

		Expect(previews(linkedReferences("target"))).To(ConsistOf("tagged #target here"))
	})

	It("finds blocks that embed the page", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md":   "- content of target\n",
			"referrer.md": "- {{embed [[target]]}}\n",
		})

		Expect(linkedReferences("target").Size()).To(Equal(1))
	})

	It("finds blocks that reference the page in a property", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md":   "- content of target\n",
			"referrer.md": "- a block\n  related:: [[target]]\n",
		})

		Expect(linkedReferences("target").Size()).To(Equal(1))
	})

	It("finds references that differ in case from the title", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md":   "- content of target\n",
			"referrer.md": "- see [[TARGET]] for more\n",
		})

		Expect(linkedReferences("target").Size()).To(Equal(1))
	})

	It("finds references in journals", func() {
		Expect(os.WriteFile(
			filepath.Join(dir, "journals", "2025_06_15.md"),
			[]byte("- see [[target]] for more\n"),
			0o644,
		)).To(Succeed())

		graph = openGraphWithPages(dir, map[string]string{
			"target.md": "- content of target\n",
		})

		results := linkedReferences("target")
		Expect(results.Size()).To(Equal(1))
		Expect(results.Results()[0].PageType()).To(Equal(logseq.PageTypeJournal))
		Expect(results.Results()[0].PageDate()).To(Equal(time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local)))
	})

	It("finds the references of a journal", func() {
		Expect(os.WriteFile(
			filepath.Join(dir, "journals", "2025_06_15.md"),
			[]byte("- journal content\n"),
			0o644,
		)).To(Succeed())

		// Journals are referenced by their title, which the default format of
		// the page title makes `Sun 15, Jun 2025` for this date.
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- see [[Sun 15, Jun 2025]] for more\n",
		})

		journal, err := graph.OpenJournal(time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local))
		Expect(err).ToNot(HaveOccurred())
		Expect(journal.Title()).To(Equal("Sun 15, Jun 2025"))

		results, err := journal.LinkedReferences(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(results.Size()).To(Equal(1))
		Expect(results.Results()[0].PageTitle()).To(Equal("referrer"))
	})

	It("includes blocks on the page itself", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md": "- target refers to [[target]]\n",
		})

		results := linkedReferences("target")
		Expect(results.Size()).To(Equal(1))
		Expect(results.Results()[0].PageTitle()).To(Equal("target"))
	})

	It("finds nothing when no block references the page", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md": "- content of target\n",
		})

		Expect(linkedReferences("target").Size()).To(BeZero())
	})

	It("finds the references of a page that does not exist yet", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- see [[missing]] for more\n",
		})

		page, err := graph.OpenPage("missing")
		Expect(err).ToNot(HaveOccurred())
		Expect(page.IsNew()).To(BeTrue())

		results, err := page.LinkedReferences(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(results.Size()).To(Equal(1))
	})

	It("reports the total number of references while returning a page of them", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md": "- content of target\n",
			"a.md":      "- a sees [[target]]\n",
			"b.md":      "- b sees [[target]]\n",
			"c.md":      "- c sees [[target]]\n",
		})

		results := linkedReferences("target", logseq.WithMaxHits(2))
		Expect(results.Size()).To(Equal(2))
		Expect(results.Count()).To(Equal(3))
	})

	It("narrows the references down with a query", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md": "- content of target\n",
			"a.md":      "- a sees [[target]] while thinking about turnips\n",
			"b.md":      "- b sees [[target]]\n",
		})

		results := linkedReferences("target", logseq.WithQuery(logseq.ContentMatches("turnips")))
		Expect(results.Size()).To(Equal(1))
		Expect(results.Results()[0].PageTitle()).To(Equal("a"))
	})

	It("opens the block that references the page", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md":   "- content of target\n",
			"referrer.md": "- see [[target]] for more\n",
		})

		results := linkedReferences("target")
		Expect(results.Size()).To(Equal(1))

		block, page, err := results.Results()[0].Open()
		Expect(err).ToNot(HaveOccurred())
		Expect(page.Title()).To(Equal("referrer"))

		ref := block.Content().FindDeep(content.IsPageReference())
		Expect(ref.(content.PageRef).GetTo()).To(Equal("target"))
	})

	It("saves changes to a reference opened in a transaction", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md":   "- content of target\n",
			"referrer.md": "- see [[target]] for more\n",
		})

		tx := graph.NewTransaction()
		page, err := tx.OpenPage("target")
		Expect(err).ToNot(HaveOccurred())

		results, err := page.LinkedReferences(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(results.Size()).To(Equal(1))

		block, _, err := results.Results()[0].Open()
		Expect(err).ToNot(HaveOccurred())
		block.AddChild(textBlock("a note about the reference"))

		Expect(tx.Save()).To(Succeed())

		data, err := os.ReadFile(filepath.Join(dir, "pages", "referrer.md"))
		Expect(err).ToNot(HaveOccurred())
		Expect(string(data)).To(Equal("- see [[target]] for more\n\t- a note about the reference\n"))
	})

	It("fails when indexing is not enabled", func() {
		var err error
		graph, err = logseq.Open(ctx, dir)
		Expect(err).ToNot(HaveOccurred())

		page, err := graph.OpenPage("target")
		Expect(err).ToNot(HaveOccurred())

		_, err = page.LinkedReferences(ctx)
		Expect(err).To(HaveOccurred())
	})
})
