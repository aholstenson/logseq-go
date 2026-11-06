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

		It("points page links with a label at the new title", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"old.md":      "- content of old\n",
				"referrer.md": "- see [the old page]([[old]]) for more\n",
			})

			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new title")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("referrer.md")).To(Equal("- see [the old page]([[new title]]) for more\n"))
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
		It("points page links inside a note block at the new title", func() {
			// Given page r with a NOTE block that links to old
			graph = openGraphWithPages(dir, map[string]string{
				"old.md": "- content of old\n",
				"r.md":   "- n\n  #+BEGIN_NOTE\n  see [[old]]\n  #+END_NOTE\n",
			})

			// When old is renamed to new
			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			// Then the link inside the NOTE block points at new
			Expect(readPage("r.md")).To(ContainSubstring("see [[new]]"))
			Expect(readPage("r.md")).ToNot(ContainSubstring("[[old]]"))
		})
		It("changes only the reference in a page that references the page", func() {
			// Given page r with a link to old and other blocks that do not
			// reference old
			graph = openGraphWithPages(dir, map[string]string{
				"old.md": "- content of old\n",
				"r.md":   "- see [[old]]\n- *see #x*\n- [1] note\n",
			})

			// When old is renamed to new
			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			// Then only the link changes and the other blocks keep their text
			Expect(readPage("r.md")).To(Equal("- see [[new]]\n- *see #x*\n- [1] note\n"))
		})
		It("points a page link nested in another page link at the new title", func() {
			// Given page r with a page link to b nested in a page link
			graph = openGraphWithPages(dir, map[string]string{
				"b.md": "- content of b\n",
				"r.md": "- [[a [[b]]]]\n",
			})

			// When page b is renamed to c
			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "b", "c")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			// Then the inner link points at c and no backslash is added
			Expect(readPage("r.md")).To(Equal("- [[a [[c]]]]\n"))
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

	Describe("Save", func() {
		It("leaves a page that was opened but not changed untouched", func() {
			// Given a page with a heading block and a table block
			original := "- ## H\n- | a |\n  |---|\n"
			graph = openGraphWithPages(dir, map[string]string{
				"p.md": original,
			})

			before, err := os.Stat(filepath.Join(dir, "pages", "p.md"))
			Expect(err).ToNot(HaveOccurred())

			// When it is opened in a transaction and saved without changes
			tx := graph.NewTransaction()
			_, err = tx.OpenPage("p")
			Expect(err).ToNot(HaveOccurred())
			Expect(tx.Save()).To(Succeed())

			// Then the file has the same bytes and modification time
			Expect(readPage("p.md")).To(Equal(original))

			after, err := os.Stat(filepath.Join(dir, "pages", "p.md"))
			Expect(err).ToNot(HaveOccurred())
			Expect(after.ModTime()).To(Equal(before.ModTime()))
		})

		It("keeps the text of the blocks that were not changed", func() {
			// Given a page where the writer would change the text of the
			// first two blocks
			graph = openGraphWithPages(dir, map[string]string{
				"p.md": "- *see #x*\n- | a |\n  |---|\n",
			})

			// When a block is added and the page is saved
			tx := graph.NewTransaction()
			page, err := tx.OpenPage("p")
			Expect(err).ToNot(HaveOccurred())
			page.AddBlock(textBlock("added"))
			Expect(tx.Save()).To(Succeed())

			// Then only the new block is added to the file
			Expect(readPage("p.md")).To(Equal("- *see #x*\n- | a |\n  |---|\n- added\n"))
		})

		It("keeps the text of the blocks that were not changed when saved again", func() {
			// Given a page that was changed and saved once
			graph = openGraphWithPages(dir, map[string]string{
				"p.md": "- *see #x*\n",
			})

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("p")
			Expect(err).ToNot(HaveOccurred())
			page.AddBlock(textBlock("first"))
			Expect(tx.Save()).To(Succeed())

			// When it is changed and saved again in the same transaction
			page.AddBlock(textBlock("second"))
			Expect(tx.Save()).To(Succeed())

			// Then the first block still has its text
			Expect(readPage("p.md")).To(Equal("- *see #x*\n- first\n- second\n"))
		})

		It("keeps the text of the renamed page in its new file", func() {
			// Given a page with a block that the writer would change
			graph = openGraphWithPages(dir, map[string]string{
				"old.md": "- *see #x*\n- see [[old]]\n",
			})

			// When the page is renamed
			tx := graph.NewTransaction()
			Expect(tx.RenamePage(ctx, "old", "new")).To(Succeed())
			Expect(tx.Save()).To(Succeed())

			// Then only the link to itself changes
			Expect(readPage("new.md")).To(Equal("- *see #x*\n- see [[new]]\n"))
		})
	})
})

var _ = Describe("Renaming a page that is named in a quoted property value", func() {
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

	It("leaves a link inside a quoted property value alone", func() {
		// Given a page with a link inside a quoted property value
		graph = openGraphWithPages(dir, map[string]string{
			"x.md":        "- content of x\n",
			"referrer.md": "- a\n  foo:: \"[[x]]\"\n",
		})

		// When the page x is renamed
		tx := graph.NewTransaction()
		Expect(tx.RenamePage(ctx, "x", "new")).To(Succeed())
		Expect(tx.Save()).To(Succeed())

		// Then the quoted value is text for Logseq and does not change
		data, err := os.ReadFile(filepath.Join(dir, "pages", "referrer.md"))
		Expect(err).ToNot(HaveOccurred())
		Expect(string(data)).To(Equal("- a\n  foo:: \"[[x]]\"\n"))
	})
})
