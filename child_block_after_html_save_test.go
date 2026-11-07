package logseq_test

import (
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// An HTML block ends only at a blank line. A child block that is added
// after an HTML block must still read back as a child block, and not as a
// part of the HTML.
var _ = Describe("Adding a child block after an HTML block", func() {
	var (
		graph *logseq.Graph
		dir   string
	)

	BeforeEach(func() {
		dir = setupGraph()
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

	It("reads the new block back as a child block", func() {
		// Given a page with a block that ends with an HTML block
		graph = openGraphWithPages(dir, map[string]string{
			"html.md": "- <div>\n  x\n",
		})

		// When a child block is added and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("html")
		Expect(err).ToNot(HaveOccurred())
		page.Blocks()[0].AddChild(content.NewBlock(content.NewParagraph(content.NewText("child"))))
		Expect(tx.Save()).To(Succeed())
		saved := readPage("html.md")

		// Then the block reads back with a child block `child`
		again, err := graph.NewTransaction().OpenPage("html")
		Expect(err).ToNot(HaveOccurred())
		Expect(again.Blocks()).To(HaveLen(1), "saved: %q", saved)
		block := again.Blocks()[0]
		Expect(block.Blocks()).To(HaveLen(1), "saved: %q", saved)
		Expect(block.Blocks()[0].Children().FindDeep(func(n content.Node) bool {
			t, ok := n.(*content.Text)
			return ok && t.Value == "child"
		})).ToNot(BeNil(), "saved: %q", saved)

		// And the HTML block does not have the text of the child block
		html := block.Children().FindDeep(content.IsOfType[*content.RawHTMLBlock]())
		Expect(html).ToNot(BeNil(), "saved: %q", saved)
		Expect(html.(*content.RawHTMLBlock).HTML).ToNot(ContainSubstring("child"), "saved: %q", saved)
	})
})
