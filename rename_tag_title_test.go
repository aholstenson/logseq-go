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

var _ = Describe("Renaming a page that is used as a tag", func() {
	var graph *logseq.Graph

	AfterEach(func() {
		if graph != nil {
			graph.Close()
			graph = nil
		}
	})

	It("writes the tag so that it points at a new title with a comma", func() {
		// Given a page `r` that tags the page `old`
		dir := setupGraph()
		graph = openGraphWithPages(dir, map[string]string{
			"old.md": "- content of old\n",
			"r.md":   "- #old\n",
		})

		// When `old` is renamed to `new,name`
		tx := graph.NewTransaction()
		Expect(tx.RenamePage(context.Background(), "old", "new,name")).To(Succeed())
		Expect(tx.Save()).To(Succeed())

		// Then the saved page `r` has a ref to `new,name`, and not to `new`
		data, err := os.ReadFile(filepath.Join(dir, "pages", "r.md"))
		Expect(err).ToNot(HaveOccurred())

		block, err := logseq.ParseBlock(string(data))
		Expect(err).ToNot(HaveOccurred())

		titles := []string{}
		for _, ref := range block.Children().PageReferences() {
			titles = append(titles, ref.(content.PageRef).GetTo())
		}
		Expect(titles).To(Equal([]string{"new,name"}), "saved text: %q", string(data))
	})

	It("keeps the bracket form of a tag when the page is renamed", func() {
		// Given a page `r` that tags the page `old` with `#[[old]]`
		dir := setupGraph()
		graph = openGraphWithPages(dir, map[string]string{
			"old.md": "- content of old\n",
			"r.md":   "- #[[old]]\n",
		})

		// When `old` is renamed to `new`
		tx := graph.NewTransaction()
		Expect(tx.RenamePage(context.Background(), "old", "new")).To(Succeed())
		Expect(tx.Save()).To(Succeed())

		// Then the tag keeps its form
		data, err := os.ReadFile(filepath.Join(dir, "pages", "r.md"))
		Expect(err).ToNot(HaveOccurred())
		Expect(string(data)).To(Equal("- #[[new]]\n"))
	})
})
