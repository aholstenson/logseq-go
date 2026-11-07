package logseq_test

import (
	"context"
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq splits the values of properties such as `tags::` and `alias::` at
// commas. When a page is renamed to a title with a comma, the new title in
// such a value must be written so that it stays one title.
var _ = Describe("Renaming a page to a title with a comma in property values", func() {
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

	rename := func(referrer string, title string) string {
		graph = openGraphWithPages(dir, map[string]string{
			"old.md":      "- content of old\n",
			"referrer.md": referrer,
		})

		tx := graph.NewTransaction()
		Expect(tx.RenamePage(ctx, "old", title)).To(Succeed())
		Expect(tx.Save()).To(Succeed())
		return readPage("referrer.md")
	}

	It("keeps a new title with a comma as one tag", func() {
		// Given a block with the tags `old` and `Other`
		// When `old` is renamed to `x, y` and the pages are saved
		saved := rename("- a\n  tags:: old, Other\n", "x, y")

		// Then the new title is in brackets
		Expect(saved).To(Equal("- a\n  tags:: [[x, y]], Other\n"))

		// And the block refers to the pages `x, y` and `Other`
		page, err := graph.NewTransaction().OpenPage("referrer")
		Expect(err).ToNot(HaveOccurred())
		titles := []string{}
		for _, ref := range page.Blocks()[0].Children().PageReferences() {
			titles = append(titles, ref.(content.PageRef).GetTo())
		}
		Expect(titles).To(ConsistOf("x, y", "Other"))
	})

	It("keeps a new title with a comma as one alias", func() {
		// Given a page with the alias `old`
		// When `old` is renamed to `New, Name` and the pages are saved
		saved := rename("alias:: old\n\n- a\n", "New, Name")

		// Then the new alias is in brackets
		Expect(saved).To(ContainSubstring("alias:: [[New, Name]]\n"))
	})

	It("writes a new title with no comma in a tag value without brackets", func() {
		// Given a block with the tags `old` and `Other`
		// When `old` is renamed to `New Name` and the pages are saved
		saved := rename("- a\n  tags:: old, Other\n", "New Name")

		// Then the new title is written as it is
		Expect(saved).To(Equal("- a\n  tags:: New Name, Other\n"))
	})
})
