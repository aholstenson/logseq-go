package logseq_test

import (
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A `$$` line without a closing line is text for Logseq. Saving a page
// with such a line after a change does not add a closing `$$` line.
var _ = Describe("Saving a page with an unclosed math line", func() {
	var graph *logseq.Graph

	AfterEach(func() {
		if graph != nil {
			graph.Close()
			graph = nil
		}
	})

	It("does not add a closing math line when a block is added at the end", func() {
		// Given a page with an unclosed `$$` line before two bullets
		dir := setupGraph()
		graph = openGraphWithPages(dir, map[string]string{
			"math.md": "text\n$$\n- a [[P]]\n- b\n",
		})

		// When a block is added at the end and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("math")
		Expect(err).ToNot(HaveOccurred())
		page.AddBlock(textBlock("NEW"))
		Expect(tx.Save()).To(Succeed())

		// Then only the new block is added to the file
		data, err := os.ReadFile(filepath.Join(dir, "pages", "math.md"))
		Expect(err).ToNot(HaveOccurred())
		Expect(string(data)).To(Equal("text\n$$\n- a [[P]]\n- b\n- NEW\n"))
	})
})
