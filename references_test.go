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

	It("finds only the block with the link, not its parents", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md":   "- content of target\n",
			"referrer.md": "- parent\n  - child\n    - see [[target]]\n  - sibling\n",
		})

		Expect(previews(linkedReferences("target"))).To(ConsistOf("see target"))
	})

	It("finds blocks that tag the page", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md":   "- content of target\n",
			"referrer.md": "- tagged #target here\n",
		})

		Expect(previews(linkedReferences("target"))).To(ConsistOf("tagged #target here"))
	})

	It("finds blocks that link to the page with a label", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target page.md": "- content of target\n",
			"referrer.md":    "- see [the target]([[target page]]) for more\n",
		})

		Expect(previews(linkedReferences("target page"))).To(ConsistOf("see the target for more"))
	})

	It("finds blocks that embed the page", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md":   "- content of target\n",
			"referrer.md": "- {{embed [[target]]}}\n",
		})

		Expect(linkedReferences("target").Size()).To(Equal(1))
	})

	It("does not find the links in queries, the same as Logseq", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"target.md": "- content of target\n",
			"referrer.md": "- simple {{query [[target]]}}\n" +
				"- tag {{query #target}}\n" +
				"- #+BEGIN_QUERY\n" +
				"  {:title \"[[target]]\" :query [:find (pull ?b [*]) :where [?b :block/refs [:block/name \"target\"]]]}\n" +
				"  #+END_QUERY\n" +
				"- see [[target]]\n",
		})

		Expect(previews(linkedReferences("target"))).To(ConsistOf("see target"))
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
		// the page title makes `Jun 15th, 2025` for this date.
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- see [[Jun 15th, 2025]] for more\n",
		})

		journal, err := graph.OpenJournal(time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local))
		Expect(err).ToNot(HaveOccurred())
		Expect(journal.Title()).To(Equal("Jun 15th, 2025"))

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

	It("finds blocks that link to a page in the namespace of the page", func() {
		// Given page r with a link to the namespace page a/b/c
		graph = openGraphWithPages(dir, map[string]string{
			"r.md": "- see [[a/b/c]]\n",
		})

		// When the linked references of the namespace parents are read
		// Then both parents have the block, the same as Logseq (block.cljs adds
		// the namespace parents of every ref to the refs of the block)
		Expect(previews(linkedReferences("a"))).To(ConsistOf("see a/b/c"))
		Expect(previews(linkedReferences("a/b"))).To(ConsistOf("see a/b/c"))
	})

	It("finds blocks that link to the page with slashes around the title", func() {
		// Given page r with a link to /a/
		graph = openGraphWithPages(dir, map[string]string{
			"r.md": "- see [[/a/]]\n",
		})

		// When the linked references of page a are read
		// Then the block is there, the same as Logseq (which removes the
		// slashes at the start and end of a page name)
		Expect(linkedReferences("a").Size()).To(Equal(1))
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

var _ = Describe("Linked references of property pages", func() {
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

	referenceCount := func(title string) int {
		page, err := graph.OpenPage(title)
		Expect(err).ToNot(HaveOccurred())

		results, err := page.LinkedReferences(ctx)
		Expect(err).ToNot(HaveOccurred())
		return results.Size()
	}

	writeConfig := func(config string) {
		Expect(os.WriteFile(filepath.Join(dir, "logseq", "config.edn"), []byte(config), 0o644)).To(Succeed())
	}

	It("links a block to the page named by its property, the same as Logseq", func() {
		// Given a block with a custom property
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- a\n  type:: book\n",
		})

		// Then the block is a linked reference of the property page, and
		// the plain text value is not a reference
		Expect(referenceCount("type")).To(Equal(1))
		Expect(referenceCount("book")).To(Equal(0))
	})

	It("does not link a block to the page of a built-in property", func() {
		// Given a block with the built-in property tags
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- a\n  tags:: x\n",
		})

		// Then the block is linked to the tag but not to a page tags
		Expect(referenceCount("tags")).To(Equal(0))
		Expect(referenceCount("x")).To(Equal(1))
	})

	It("links a block with a namespaced property to the page of the last part of the name", func() {
		// Given a block with the property ns/key
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- a\n  ns/key:: v\n",
		})

		// Then the block refs the page key, the same as Logseq
		Expect(referenceCount("key")).To(Equal(1))
	})

	It("links the page properties to the property page", func() {
		// Given page properties before the first block
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "type:: book\n\n- a\n",
		})

		// Then the pre-block is a linked reference of the property page
		Expect(referenceCount("type")).To(Equal(1))
	})

	It("does not link to property pages when the graph turns them off", func() {
		// Given a graph with :property-pages/enabled? false
		writeConfig("{:property-pages/enabled? false}")
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- a\n  type:: book\n",
		})

		// Then no reference is made from the property name
		Expect(referenceCount("type")).To(Equal(0))
	})

	It("does not link to a property page that the graph excludes", func() {
		// Given a graph that excludes the property type
		writeConfig("{:property-pages/excludelist [:type]}")
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- a\n  type:: book\n  status:: open\n",
		})

		// Then type makes no reference, but other names still do
		Expect(referenceCount("type")).To(Equal(0))
		Expect(referenceCount("status")).To(Equal(1))
	})

	It("does not link to a page named by an excluded namespaced property", func() {
		// Given a graph that excludes the property key in a namespace
		writeConfig("{:property-pages/excludelist [:ns/key]}")
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- a\n  ns/key:: v\n",
		})

		// Then the name of the property makes no reference
		Expect(referenceCount("key")).To(Equal(0))
	})

	It("does not link to a page named by an upper-case built-in property", func() {
		// Given a block with the built-in property collapsed, written in
		// upper case
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- a\n  Collapsed:: true\n",
		})

		// Then the name of the property makes no reference
		Expect(referenceCount("collapsed")).To(Equal(0))
	})

	It("puts the property page in the path refs of child blocks", func() {
		// Given a child block below a block with a custom property
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- a\n  type:: book\n\t- child\n",
		})

		// Then both blocks have the property page in their path refs
		results, err := graph.SearchBlocks(ctx, logseq.WithQuery(logseq.PathReferences("type")))
		Expect(err).ToNot(HaveOccurred())
		Expect(results.Size()).To(Equal(2))
	})
})

var _ = Describe("Linked references of task markers and priorities", func() {
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

	referenceCount := func(title string) int {
		page, err := graph.OpenPage(title)
		Expect(err).ToNot(HaveOccurred())

		results, err := page.LinkedReferences(ctx)
		Expect(err).ToNot(HaveOccurred())
		return results.Size()
	}

	It("links a task to the page of its marker, the same as Logseq", func() {
		// Given a TODO task
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- TODO buy milk\n",
		})

		// Then the task is a linked reference of the page TODO
		Expect(referenceCount("TODO")).To(Equal(1))
	})

	It("links a task to the pages of its marker and its priority", func() {
		// Given a DONE task with priority B
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- DONE [#B] x\n",
		})

		// Then the task is a linked reference of the pages DONE and B
		Expect(referenceCount("DONE")).To(Equal(1))
		Expect(referenceCount("B")).To(Equal(1))
	})

	It("links a task with marker and priority A to both pages", func() {
		// Given a TODO task with priority A
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- TODO [#A] task\n",
		})

		// Then the task is a linked reference of the pages TODO and A
		Expect(referenceCount("TODO")).To(Equal(1))
		Expect(referenceCount("A")).To(Equal(1))
	})

	It("does not link to the page TODO when the word is not at the start", func() {
		// Given a block with TODO in the middle of its text
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- buy TODO milk\n",
		})

		// Then the block is not a linked reference of the page TODO
		Expect(referenceCount("TODO")).To(Equal(0))
	})

	It("links a block with a priority and no marker to the page of the priority", func() {
		// Given a block that starts with a priority
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- [#C] x\n",
		})

		// Then the block is a linked reference of the page C
		Expect(referenceCount("C")).To(Equal(1))
	})

	It("puts the task marker in the path refs of child blocks", func() {
		// Given a child block below a task
		graph = openGraphWithPages(dir, map[string]string{
			"referrer.md": "- TODO x\n\t- child\n",
		})

		// Then both blocks have the page TODO in their path refs
		results, err := graph.SearchBlocks(ctx, logseq.WithQuery(logseq.PathReferences("TODO")))
		Expect(err).ToNot(HaveOccurred())
		Expect(results.Size()).To(Equal(2))
	})
})
