package logseq_test

import (
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The first block of a page can have only an `id::` property, because
// another block refers to it. An edit in another block of the page must not
// change the first block into page properties, as the block reference would
// then point at no block.
var _ = Describe("Saving a page whose first block has only an id", func() {
	const id = "650c9a1c-6a2b-4b0c-9a77-1f2d3e4f5a6b"

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

	It("keeps the first block as a block with the id when another block is edited", func() {
		// Given a page whose first block has only an id, and a second block that refers to it
		graph = openGraphWithPages(dir, map[string]string{
			"ids.md": "- id:: " + id + "\n- b ((" + id + "))\n",
		})

		// When the text of the second block is changed and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("ids")
		Expect(err).ToNot(HaveOccurred())
		text := page.Blocks()[1].Children().FindDeep(content.IsOfType[*content.Text]())
		Expect(text).ToNot(BeNil())
		text.(*content.Text).WithValue("c ")
		Expect(tx.Save()).To(Succeed())
		saved := readPage("ids.md")

		// Then the first block is still a block with the id
		again, err := graph.NewTransaction().OpenPage("ids")
		Expect(err).ToNot(HaveOccurred())
		first := again.Blocks()[0]
		Expect(first.IsPreBlock()).To(BeFalse(), "saved: %q", saved)
		Expect(first.ID()).To(Equal(id), "saved: %q", saved)
		Expect(saved).To(Equal("- id:: " + id + "\n- c ((" + id + "))\n"))
	})
})
