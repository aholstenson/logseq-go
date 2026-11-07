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

// A page can be renamed to a title with characters that end a part of a
// link, such as an unbalanced `)`. The links to the page must still read
// back as links to the new title after the rename is saved.
var _ = Describe("Renaming a page to a title with a parenthesis", func() {
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

	// renameAndReadLink renames the page `old` and gives the first page
	// link of the page `referrer` when it is read again.
	renameAndReadLink := func(referrer string, title string) (*content.PageLink, string) {
		graph = openGraphWithPages(dir, map[string]string{
			"old.md":      "- content of old\n",
			"referrer.md": referrer,
		})

		tx := graph.NewTransaction()
		Expect(tx.RenamePage(ctx, "old", title)).To(Succeed())
		Expect(tx.Save()).To(Succeed())
		saved := readPage("referrer.md")

		page, err := graph.NewTransaction().OpenPage("referrer")
		Expect(err).ToNot(HaveOccurred())
		for _, block := range page.Blocks() {
			if link := block.Children().FindDeep(content.IsOfType[*content.PageLink]()); link != nil {
				return link.(*content.PageLink), saved
			}
		}

		return nil, saved
	}

	labelOf := func(link *content.PageLink) string {
		label := ""
		for _, n := range link.Children().FilterDeep(content.IsOfType[*content.Text]()) {
			label += n.(*content.Text).Value
		}

		return label
	}

	It("keeps a labelled page link when the new title has an unbalanced closing parenthesis", func() {
		// Given a page with a labelled page link to `old`
		// When `old` is renamed to `a) b` and the pages are saved and read again
		link, saved := renameAndReadLink("- [x]([[old]])\n", "a) b")

		// Then the block has a page link to `a) b`
		Expect(link).ToNot(BeNil(), "saved: %q", saved)
		Expect(link.To).To(Equal("a) b"), "saved: %q", saved)
	})

	It("keeps the label of a page link when the new title has a closing parenthesis after a space", func() {
		// Given a page with a labelled page link to `old` in a line of text
		// When `old` is renamed to `a ) b` and the pages are saved and read again
		link, saved := renameAndReadLink("- see [l]([[old]]) x\n", "a ) b")

		// Then the block has a page link to `a ) b` with the label `l`
		Expect(link).ToNot(BeNil(), "saved: %q", saved)
		Expect(link.To).To(Equal("a ) b"), "saved: %q", saved)
		Expect(labelOf(link)).To(Equal("l"), "saved: %q", saved)
	})

	It("writes a title with balanced parentheses in a labelled page link as it is", func() {
		// Given a page with a labelled page link to `old`
		// When `old` is renamed to `Meeting (2024)` and the pages are saved
		_, saved := renameAndReadLink("- [x]([[old]])\n", "Meeting (2024)")

		// Then the link has the new title with no change
		Expect(saved).To(Equal("- [x]([[Meeting (2024)]])\n"))
	})
})
