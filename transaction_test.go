package logseq_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// textBlock creates a block with a single paragraph of text in it.
func textBlock(text string) *content.Block {
	return content.NewBlock(content.NewParagraph(content.NewText(text)))
}

var _ = Describe("Transaction", func() {
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

	readPage := func(name string) string {
		data, err := os.ReadFile(filepath.Join(dir, "pages", name))
		Expect(err).ToNot(HaveOccurred())
		return string(data)
	}

	Describe("RenamePage", func() {
		It("moves the page to the file of the new title", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md": "- content of old\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("new.md")).To(Equal("- content of old\n"))
			Expect(filepath.Join(dir, "pages", "old.md")).ToNot(BeAnExistingFile())
		})

		It("points page links at the new title", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md":       "- content of old\n",
				"referrer.md":  "- see [[old]] for more\n",
				"unrelated.md": "- see [[other]] for more\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("referrer.md")).To(Equal("- see [[new]] for more\n"))
			Expect(readPage("unrelated.md")).To(Equal("- see [[other]] for more\n"))
		})

		It("points hashtags at the new title", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md":      "- content of old\n",
				"referrer.md": "- tagged #old here\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("referrer.md")).To(Equal("- tagged #new here\n"))
		})

		It("uses the extended hashtag syntax when the new title has spaces", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md":      "- content of old\n",
				"referrer.md": "- tagged #old here\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new title")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("referrer.md")).To(Equal("- tagged #[[new title]] here\n"))
		})

		It("points page embeds at the new title", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md":      "- content of old\n",
				"referrer.md": "- {{embed [[old]]}}\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("referrer.md")).To(Equal("- {{embed [[new]]}}\n"))
		})

		It("points references in properties at the new title", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md":      "- content of old\n",
				"referrer.md": "related:: [[old]]\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("referrer.md")).To(Equal("related:: [[new]]\n"))
		})

		It("points references that differ in case at the new title", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md":      "- content of old\n",
				"referrer.md": "- see [[Old]] for more\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("referrer.md")).To(Equal("- see [[new]] for more\n"))
		})

		It("points references on the renamed page itself at the new title", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md": "- old refers to [[old]]\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("new.md")).To(Equal("- old refers to [[new]]\n"))
			Expect(filepath.Join(dir, "pages", "old.md")).ToNot(BeAnExistingFile())
		})

		It("points references in journals at the new title", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "journals", "2025_06_15.md"),
				[]byte("- see [[old]] for more\n"),
				0o644,
			)).To(Succeed())

			graph = openGraphWithPages(dir, map[string]string{
				"old.md": "- content of old\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			data, err := os.ReadFile(filepath.Join(dir, "journals", "2025_06_15.md"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("- see [[new]] for more\n"))
		})

		It("leaves pages that do not reference the page untouched", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md":       "- content of old\n",
				"unrelated.md": "Prose that   would be reformatted\n- see [[other]]\n",
			})

			before, err := os.Stat(filepath.Join(dir, "pages", "unrelated.md"))
			Expect(err).ToNot(HaveOccurred())

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			after, err := os.Stat(filepath.Join(dir, "pages", "unrelated.md"))
			Expect(err).ToNot(HaveOccurred())
			Expect(after.ModTime()).To(Equal(before.ModTime()))
		})

		It("keeps changes made to the page in the same transaction", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md": "- content of old\n",
			})

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("old")
			Expect(err).ToNot(HaveOccurred())
			page.AddBlock(textBlock("added later"))

			Expect(tx.RenamePage(ctx, "old", "new")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("new.md")).To(Equal("- content of old\n- added later\n"))
		})

		It("renames a page that only changes in case", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md":      "- content of old\n",
				"referrer.md": "- see [[old]] for more\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "Old")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("referrer.md")).To(Equal("- see [[Old]] for more\n"))

			page, err := graph.OpenPage("Old")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.IsNew()).To(BeFalse())
			Expect(page.Blocks()).To(HaveLen(1))
		})

		It("leaves the pages in the namespace of the page alone", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md":        "- content of old\n",
				"old___kept.md": "- content of kept\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("old___kept.md")).To(Equal("- content of kept\n"))
		})

		It("fails when the title is an alias of another page", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "alias:: Other\n- content of target\n",
			})

			tx := graph.NewTransaction()
			err := tx.RenamePage(ctx, "Other", "new")
			Expect(errors.Is(err, logseq.ErrPageNotFound)).To(BeTrue())

			Expect(tx.Save()).To(Succeed())
			Expect(readPage("target.md")).To(Equal("alias:: Other\n- content of target\n"))
			Expect(filepath.Join(dir, "pages", "new.md")).ToNot(BeAnExistingFile())
		})

		It("fails when the page does not exist", func() {
			graph = openGraphWithPages(dir, map[string]string{})

			tx := graph.NewTransaction()
			err := tx.RenamePage(ctx, "missing", "new")
			Expect(errors.Is(err, logseq.ErrPageNotFound)).To(BeTrue())
		})

		It("fails when the new title is taken", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md": "- content of old\n",
				"new.md": "- content of new\n",
			})

			tx := graph.NewTransaction()
			err := tx.RenamePage(ctx, "old", "new")
			Expect(errors.Is(err, logseq.ErrPageExists)).To(BeTrue())

			// Neither page was touched
			Expect(tx.Save()).To(Succeed())
			Expect(readPage("old.md")).To(Equal("- content of old\n"))
			Expect(readPage("new.md")).To(Equal("- content of new\n"))
		})

		It("fails when indexing is not enabled", func() {
			var err error
			graph, err = logseq.Open(ctx, dir)
			Expect(err).ToNot(HaveOccurred())

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new")).ToNot(Succeed())
		})
	})

	Describe("RenamePage with namespace children", func() {
		It("renames the pages in the namespace of the page", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md":         "- content of old\n",
				"old___child.md": "- content of child\n",
				"old___other.md": "- content of other\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new", logseq.WithNamespaceChildren())).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("new.md")).To(Equal("- content of old\n"))
			Expect(readPage("new___child.md")).To(Equal("- content of child\n"))
			Expect(readPage("new___other.md")).To(Equal("- content of other\n"))

			Expect(filepath.Join(dir, "pages", "old___child.md")).ToNot(BeAnExistingFile())
			Expect(filepath.Join(dir, "pages", "old___other.md")).ToNot(BeAnExistingFile())
		})

		It("renames the pages deeper in the namespace", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md":                      "- content of old\n",
				"old___child.md":              "- content of child\n",
				"old___child___nested.md":     "- content of nested\n",
				"old___child___nested___x.md": "- content of x\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new", logseq.WithNamespaceChildren())).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("new___child___nested.md")).To(Equal("- content of nested\n"))
			Expect(readPage("new___child___nested___x.md")).To(Equal("- content of x\n"))
		})

		It("points references to the renamed pages at their new titles", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md":         "- content of old\n",
				"old___child.md": "- content of child\n",
				"referrer.md":    "- see [[old]] and [[old/child]] for more\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new", logseq.WithNamespaceChildren())).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("referrer.md")).To(Equal("- see [[new]] and [[new/child]] for more\n"))
		})

		It("renames a namespace without a page of its own", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old___child.md": "- content of child\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new", logseq.WithNamespaceChildren())).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("new___child.md")).To(Equal("- content of child\n"))
			Expect(filepath.Join(dir, "pages", "old___child.md")).ToNot(BeAnExistingFile())
			Expect(filepath.Join(dir, "pages", "new.md")).ToNot(BeAnExistingFile())
		})

		It("keeps the part of the title below the namespace as it is", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"Old___Child.md": "- content of child\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new", logseq.WithNamespaceChildren())).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("new___Child.md")).To(Equal("- content of child\n"))
		})

		It("fails when neither the page nor its namespace exists", func() {
			graph = openGraphWithPages(dir, map[string]string{})

			tx := graph.NewTransaction()
			err := tx.RenamePage(ctx, "missing", "new", logseq.WithNamespaceChildren())
			Expect(errors.Is(err, logseq.ErrPageNotFound)).To(BeTrue())
		})

		It("fails when the new title of a page in the namespace is taken", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md":         "- content of old\n",
				"old___child.md": "- content of child\n",
				"new___child.md": "- content of the other child\n",
			})

			tx := graph.NewTransaction()
			err := tx.RenamePage(ctx, "old", "new", logseq.WithNamespaceChildren())
			Expect(errors.Is(err, logseq.ErrPageExists)).To(BeTrue())

			// Nothing was renamed, so the namespace is not left half renamed
			Expect(tx.Save()).To(Succeed())
			Expect(readPage("old.md")).To(Equal("- content of old\n"))
			Expect(readPage("old___child.md")).To(Equal("- content of child\n"))
			Expect(filepath.Join(dir, "pages", "new.md")).ToNot(BeAnExistingFile())
		})
	})

	Describe("DeletePage", func() {
		It("removes the page from the graph", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"gone.md": "- content\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.DeletePage("gone")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(filepath.Join(dir, "pages", "gone.md")).ToNot(BeAnExistingFile())
		})

		It("leaves references to the page alone", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"gone.md":     "- content\n",
				"referrer.md": "- see [[gone]] for more\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.DeletePage("gone")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("referrer.md")).To(Equal("- see [[gone]] for more\n"))
		})

		It("does nothing when the page does not exist", func() {
			graph = openGraphWithPages(dir, map[string]string{})

			tx := graph.NewTransaction()
			Expect(tx.DeletePage("missing")).To(Succeed())
			Expect(tx.Save()).To(Succeed())
		})

		It("drops changes made to the page in the same transaction", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"gone.md": "- content\n",
			})

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("gone")
			Expect(err).ToNot(HaveOccurred())
			page.AddBlock(textBlock("added later"))

			Expect(tx.DeletePage("gone")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(filepath.Join(dir, "pages", "gone.md")).ToNot(BeAnExistingFile())
		})

		It("moves the page to the recycle directory when recycling is enabled", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "gone.md"),
				[]byte("- content\n"),
				0o644,
			)).To(Succeed())

			var err error
			graph, err = logseq.Open(ctx, dir, logseq.WithRecycleDeletedPages())
			Expect(err).ToNot(HaveOccurred())

			tx := graph.NewTransaction()
			Expect(tx.DeletePage("gone")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(filepath.Join(dir, "pages", "gone.md")).ToNot(BeAnExistingFile())

			data, err := os.ReadFile(filepath.Join(dir, "logseq", ".recycle", "gone.md"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("- content\n"))
		})
	})

	Describe("DeleteJournal", func() {
		It("removes the journal from the graph", func() {
			date := time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local)
			Expect(os.WriteFile(
				filepath.Join(dir, "journals", "2025_06_15.md"),
				[]byte("- journal content\n"),
				0o644,
			)).To(Succeed())

			var err error
			graph, err = logseq.Open(ctx, dir)
			Expect(err).ToNot(HaveOccurred())

			tx := graph.NewTransaction()
			Expect(tx.DeleteJournal(date)).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(filepath.Join(dir, "journals", "2025_06_15.md")).ToNot(BeAnExistingFile())
		})
	})
})
