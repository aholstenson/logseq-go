package logseq_test

import (
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq reads a `\` at the end of a line as text, and two spaces as a hard
// line break. A hard line break that is added with the API must be written
// with two spaces, so that Logseq does not show a backslash.
var _ = Describe("Saving a hard line break that is added with the API", func() {
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

	It("writes a new hard line break in a quote as two spaces", func() {
		// Given a page with a quote of two lines
		graph = openGraphWithPages(dir, map[string]string{
			"quote.md": "- > a\n  > b\n",
		})

		// When the first line gets a hard line break and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("quote")
		Expect(err).ToNot(HaveOccurred())
		text := page.Blocks()[0].Children().FindDeep(func(n content.Node) bool {
			t, ok := n.(*content.Text)
			return ok && t.Value == "a"
		})
		Expect(text).ToNot(BeNil())
		text.(*content.Text).WithHardLineBreak()
		Expect(tx.Save()).To(Succeed())
		saved := readPage("quote.md")

		// Then the line of `a` ends with two spaces
		Expect(saved).To(Equal("- > a  \n  > b\n"))

		// And it does not end with a backslash
		Expect(saved).ToNot(ContainSubstring("a\\\n"))
	})
})
