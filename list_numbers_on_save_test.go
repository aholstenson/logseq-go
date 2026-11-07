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

// A block can start with a number and a dot, such as `- 2. second`. Saving
// the page after a change on that line keeps the number.
var _ = Describe("Saving a block that starts with a list number", func() {
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

	readListNumberPage := func(name string) string {
		data, err := os.ReadFile(filepath.Join(dir, "pages", name))
		Expect(err).ToNot(HaveOccurred())
		return string(data)
	}

	It("keeps the number when the text of the block is changed", func() {
		// Given a page whose first block starts with "2. "
		graph = openGraphWithPages(dir, map[string]string{
			"numbers.md": "- 2. second\n- other\n",
		})

		// When the text "second" is changed to "secondX" and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("numbers")
		Expect(err).ToNot(HaveOccurred())

		text := page.Blocks()[0].Children().FindDeep(func(node content.Node) bool {
			t, ok := node.(*content.Text)
			return ok && t.Value == "second"
		})
		Expect(text).ToNot(BeNil())
		text.(*content.Text).WithValue("secondX")
		Expect(tx.Save()).To(Succeed())

		// Then the number of the block is still 2
		Expect(readListNumberPage("numbers.md")).To(Equal("- 2. secondX\n- other\n"))
	})

	It("keeps the number when a link on the line is renamed", func() {
		// Given a page with a block that starts with "3. " and links to Old
		graph = openGraphWithPages(dir, map[string]string{
			"Old.md":  "- content of old\n",
			"list.md": "- 3. see [[Old]]\n",
		})

		// When the page Old is renamed to New and the pages are saved
		tx := graph.NewTransaction()
		Expect(tx.RenamePage(context.Background(), "Old", "New")).To(Succeed())
		Expect(tx.Save()).To(Succeed())

		// Then the number of the block is still 3
		Expect(readListNumberPage("list.md")).To(Equal("- 3. see [[New]]\n"))
	})
})
