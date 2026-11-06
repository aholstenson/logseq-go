package logseq_test

import (
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A page file can use Windows line endings (CRLF). Saving the page keeps the
// line endings of the file.
var _ = Describe("Saving pages with Windows line endings", func() {
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

	// Given a page with CRLF line endings, When it is opened and saved
	// without changes, Then the bytes of the file are the same.
	It("keeps the bytes of a page with CRLF line endings that is opened and saved without changes", func() {
		page := "- a\r\n\t- TODO b [[x]]\r\n\t  SCHEDULED: <2024-01-01 Mon>\r\n"
		graph = openGraphWithPages(dir, map[string]string{
			"crlf.md": page,
		})

		tx := graph.NewTransaction()
		_, err := tx.OpenPage("crlf")
		Expect(err).ToNot(HaveOccurred())
		Expect(tx.Save()).To(Succeed())

		Expect(readPage("crlf.md")).To(Equal(page))
	})

	// Given a page with CRLF line endings, When a block is added and the
	// page is saved, Then the lines that were in the file keep CRLF.
	It("keeps CRLF line endings of a page when a block is added", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"crlf.md": "- a\r\n\t- b\r\n",
		})

		tx := graph.NewTransaction()
		page, err := tx.OpenPage("crlf")
		Expect(err).ToNot(HaveOccurred())
		page.AddBlock(textBlock("c"))
		Expect(tx.Save()).To(Succeed())

		Expect(readPage("crlf.md")).To(HavePrefix("- a\r\n\t- b\r\n"))
	})

	// Given a page with CRLF line endings, When a block is added and the
	// page is saved, Then every line of the file ends with CRLF.
	It("writes an added block of a page with CRLF line endings with CRLF", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"crlf.md": "- a\r\n\t- b\r\n",
		})

		tx := graph.NewTransaction()
		page, err := tx.OpenPage("crlf")
		Expect(err).ToNot(HaveOccurred())
		page.AddBlock(textBlock("c"))
		Expect(tx.Save()).To(Succeed())

		Expect(readPage("crlf.md")).To(Equal("- a\r\n\t- b\r\n- c\r\n"))
	})
})
