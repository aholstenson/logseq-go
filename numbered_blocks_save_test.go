package logseq_test

import (
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Users often number their blocks, for example `- 1. foo`, `- 2. bar`. The
// number is part of the text of the block in Logseq. Saving a page after an
// edit must keep the numbers of the edited block and of the blocks next to
// it.
var _ = Describe("Saving numbered blocks after an edit", func() {
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

	findText := func(page logseq.Page, value string) *content.Text {
		for _, block := range page.Blocks() {
			for _, node := range block.Children().FilterDeep(content.IsOfType[*content.Text]()) {
				if text := node.(*content.Text); text.Value == value {
					return text
				}
			}
		}

		Fail("no text " + value)
		return nil
	}

	It("keeps the start number of a list on the bullet line when that line is changed", func() {
		// Given a page with a block that starts with a list that starts at 3
		graph = openGraphWithPages(dir, map[string]string{
			"numbers.md": "- 3. x\n  4. y\n",
		})

		// When the text of the first item is changed and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("numbers")
		Expect(err).ToNot(HaveOccurred())
		findText(page, "x").WithValue("X")
		Expect(tx.Save()).To(Succeed())

		// Then the number 3 is kept
		Expect(readPage("numbers.md")).To(Equal("- 3. X\n  4. y\n"))
	})

	It("does not change the numbers of the blocks next to an edited numbered block", func() {
		// Given a page with three numbered blocks
		graph = openGraphWithPages(dir, map[string]string{
			"numbers.md": "- 1. foo\n- 2. bar\n- 3. baz\n",
		})

		// When the text of the second block is changed and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("numbers")
		Expect(err).ToNot(HaveOccurred())
		findText(page, "bar").WithValue("bar!")
		Expect(tx.Save()).To(Succeed())

		// Then only the text of the second block is changed
		saved := readPage("numbers.md")
		Expect(saved).To(Equal("- 1. foo\n- 2. bar!\n- 3. baz\n"))

		// And the line of the third block is not changed
		Expect(saved).To(ContainSubstring("- 3. baz\n"))
	})
})
