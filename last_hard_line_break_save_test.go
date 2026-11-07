package logseq_test

import (
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A hard line break has a meaning only before more text. When the last
// text of a block or a quote has a hard line break, the break must not be
// written: a `\` at the end of the content is read as text, and a blank
// line is added after it.
var _ = Describe("Saving a hard line break on the last text of a block", func() {
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

		return nil
	}

	It("does not write a backslash when the text after a backslash break is removed", func() {
		// Given a page with a block of two lines with a backslash break between them
		graph = openGraphWithPages(dir, map[string]string{
			"hard.md": "- a\\\n  b\n- c\n",
		})

		// When the text of the second line is removed and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("hard")
		Expect(err).ToNot(HaveOccurred())
		b := findText(page, "b")
		Expect(b).ToNot(BeNil())
		b.RemoveSelf()
		Expect(tx.Save()).To(Succeed())
		saved := readPage("hard.md")

		// Then the block has only the first line
		Expect(saved).To(Equal("- a\n- c\n"))

		// And the read-back has no text with a backslash at its end
		again, err := graph.NewTransaction().OpenPage("hard")
		Expect(err).ToNot(HaveOccurred())
		Expect(findText(again, "a\\")).To(BeNil(), "saved: %q", saved)
	})

	It("does not write a hard line break on the last text of a quote", func() {
		// Given a page with a block with a quote and a second block
		graph = openGraphWithPages(dir, map[string]string{
			"hard.md": "- > a\n- b\n",
		})

		// When the text of the quote gets a hard line break and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("hard")
		Expect(err).ToNot(HaveOccurred())
		a := findText(page, "a")
		Expect(a).ToNot(BeNil())
		a.WithHardLineBreak()
		Expect(tx.Save()).To(Succeed())

		// Then the text of the page is not changed
		Expect(readPage("hard.md")).To(Equal("- > a\n- b\n"))
	})
})
