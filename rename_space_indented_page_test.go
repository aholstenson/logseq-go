package logseq_test

import (
	"context"
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Save keeps the lines of a page that did not change. A rename that changes
// one line of a page with space indentation must not change the
// indentation of the lines next to it.
var _ = Describe("Renaming a page that is linked from a space-indented page", func() {
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

	It("changes only the line with the link", func() {
		// Given a page with two-space indentation and a link to `old` in one child block
		graph = openGraphWithPages(dir, map[string]string{
			"old.md":      "- content of old\n",
			"referrer.md": "- p\n  - see [[old]]\n  - q\n",
		})

		// When `old` is renamed to `new` and the pages are saved
		tx := graph.NewTransaction()
		Expect(tx.RenamePage(context.Background(), "old", "new")).To(Succeed())
		Expect(tx.Save()).To(Succeed())

		// Then only the line with the link is changed, and the line `q` keeps its spaces
		Expect(readPage("referrer.md")).To(Equal("- p\n  - see [[new]]\n  - q\n"))
	})
})
