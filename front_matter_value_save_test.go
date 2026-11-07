package logseq_test

import (
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Front matter is YAML, not Markdown. A value that is set with the API must
// be written without Markdown escapes, as Logseq and other YAML readers
// keep the backslashes as part of the value.
var _ = Describe("Saving a front matter value that is set with the API", func() {
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

	It("writes Markdown characters in the value without backslashes", func() {
		// Given a page with front matter
		graph = openGraphWithPages(dir, map[string]string{
			"front.md": "---\ntitle: Old\n---\n- x\n",
		})

		// When the title is set to `a *b* <c>` and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("front")
		Expect(err).ToNot(HaveOccurred())
		page.Properties().Set("title", content.NewText("a *b* <c>"))
		Expect(tx.Save()).To(Succeed())
		saved := readPage("front.md")

		// Then the value is written as it is
		Expect(saved).To(Equal("---\ntitle: a *b* <c>\n---\n- x\n"))

		// And the file has no backslash
		Expect(saved).ToNot(ContainSubstring(`\`))
	})
})
