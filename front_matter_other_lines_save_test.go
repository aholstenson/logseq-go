package logseq_test

import (
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// When one property of the front matter of a page is changed, only its line
// must change. The other lines of the front matter must be kept as they are
// in the file.
var _ = Describe("Saving a change of one front matter property", func() {
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

	It("keeps the spacing of a line that is not changed", func() {
		// Given a page with front matter, and the title has three spaces after its colon
		graph = openGraphWithPages(dir, map[string]string{
			"front.md": "---\ntitle:   Old\nalias: y\n---\n- x\n",
		})

		// When the alias is changed and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("front")
		Expect(err).ToNot(HaveOccurred())
		page.Properties().Set("alias", content.NewText("z"))
		Expect(tx.Save()).To(Succeed())

		// Then only the line of the alias is changed
		Expect(readPage("front.md")).To(Equal("---\ntitle:   Old\nalias: z\n---\n- x\n"))
	})
})
