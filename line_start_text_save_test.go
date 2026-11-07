package logseq_test

import (
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The text before the first block of a page (the pre-block) can be changed
// by a program. When the new text looks like block syntax, such as `- b` or
// `$$`, the saved page must still have the same blocks.
var _ = Describe("Saving text that looks like block syntax before the first block", func() {
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

	// saveWithText opens the page, sets the text `abc` to the value and
	// saves the page.
	saveWithText := func(value string) {
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("pre")
		Expect(err).ToNot(HaveOccurred())
		findText(page, "abc").WithValue(value)
		Expect(tx.Save()).To(Succeed())
	}

	// reopen reads the saved page again.
	reopen := func() logseq.Page {
		page, err := graph.NewTransaction().OpenPage("pre")
		Expect(err).ToNot(HaveOccurred())
		return page
	}

	It("keeps a value `- b` in the text before the first block", func() {
		// Given a page with page properties, a paragraph and a block
		graph = openGraphWithPages(dir, map[string]string{
			"pre.md": "title:: x\n\nabc\n\n- c\n",
		})

		// When the paragraph text is set to `- b` and the page is saved
		saveWithText("- b")
		saved := readPage("pre.md")

		// Then the page has two blocks: the text before the first block and `c`
		blocks := reopen().Blocks()
		Expect(blocks).To(HaveLen(2), "saved: %q", saved)
		Expect(blocks[0].IsPreBlock()).To(BeTrue(), "saved: %q", saved)

		// And the first part still has the text `- b`
		texts := blocks[0].Children().FilterDeep(func(n content.Node) bool {
			t, ok := n.(*content.Text)
			return ok && t.Value == "- b"
		})
		Expect(texts).To(HaveLen(1), "saved: %q", saved)
	})

	It("does not let a value `$$` take the rest of the page", func() {
		// Given a page with page properties, a paragraph and a block
		graph = openGraphWithPages(dir, map[string]string{
			"pre.md": "title:: x\n\nabc\n\n- c\n",
		})

		// When the paragraph text is set to `$$` and the page is saved
		saveWithText("$$")
		saved := readPage("pre.md")

		// Then the page still has the block `c`
		page := reopen()
		Expect(page.Blocks().FindDeep(func(b *content.Block) bool {
			return b.Children().FindDeep(func(n content.Node) bool {
				t, ok := n.(*content.Text)
				return ok && t.Value == "c"
			}) != nil && !b.IsPreBlock()
		})).ToNot(BeNil(), "saved: %q", saved)

		// And the page has no math block
		for _, block := range page.Blocks() {
			Expect(block.Children().FindDeep(content.IsOfType[*content.MathBlock]())).To(BeNil(), "saved: %q", saved)
		}
	})
})
