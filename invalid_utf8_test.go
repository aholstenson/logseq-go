package logseq_test

import (
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A page file can hold bytes that are not valid UTF-8, for example a file
// saved as Latin-1. Saving such a page must not change those bytes without
// telling the caller.
var _ = Describe("Saving pages with bytes that are not valid UTF-8", func() {
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

	// Given a page with bytes that are not valid UTF-8, When it is opened
	// and saved without changes, Then the bytes are the same or Save
	// returns an error.
	It("keeps the bytes of a page with invalid UTF-8 that is opened and saved without changes", func() {
		page := "- a\xff\xfeb [[p\xc3]] #t\xe2\x82\n- c\n"
		graph = openGraphWithPages(dir, map[string]string{
			"latin.md": page,
		})

		tx := graph.NewTransaction()
		_, err := tx.OpenPage("latin")
		Expect(err).ToNot(HaveOccurred())
		saveErr := tx.Save()

		data, err := os.ReadFile(filepath.Join(dir, "pages", "latin.md"))
		Expect(err).ToNot(HaveOccurred())
		if saveErr == nil {
			Expect(data).To(Equal([]byte(page)))
		}
	})

	// Given a page with bytes that are not valid UTF-8, When a block is
	// added and the page is saved, Then the bytes of the other blocks are the
	// same.
	It("keeps the bytes of a page with invalid UTF-8 when a block is added", func() {
		graph = openGraphWithPages(dir, map[string]string{
			"latin.md": "- a\xff\xfeb [[p\xc3]] #t\xe2\x82\n- c\n",
		})

		tx := graph.NewTransaction()
		page, err := tx.OpenPage("latin")
		Expect(err).ToNot(HaveOccurred())
		page.AddBlock(textBlock("d"))
		Expect(tx.Save()).To(Succeed())

		data, err := os.ReadFile(filepath.Join(dir, "pages", "latin.md"))
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal([]byte("- a\xff\xfeb [[p\xc3]] #t\xe2\x82\n- c\n- d\n")))
	})
})
