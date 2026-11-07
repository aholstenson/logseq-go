package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// unclosedBlockPageText reads a page and writes it as a page is saved: the
// writer output with a line break at the end.
func unclosedBlockPageText(input string) string {
	root, err := markdown.ParseString(input, markdown.AsPage())
	Expect(err).ToNot(HaveOccurred())
	v, err := markdown.AsString(root)
	Expect(err).ToNot(HaveOccurred())
	if v != "" && !strings.HasSuffix(v, "\n") {
		v += "\n"
	}
	return v
}

// pageRefTitlesOf gets the titles of the page refs in a block.
func pageRefTitlesOf(block *content.Block) []string {
	titles := []string{}
	for _, ref := range block.Children().PageReferences() {
		titles = append(titles, ref.(content.PageRef).GetTo())
	}
	return titles
}

// A `$$` line or a code fence in a quote without a closing line is text for
// Logseq, and an HTML block at column 0 does not take in the bullets after
// it. Reading such a page keeps the blocks after the line, and saving it
// does not add a closing line.
var _ = Describe("Parsing then outputting unclosed math, fence and HTML blocks", func() {
	Describe("in a page", func() {
		// Given page text with an unclosed `$$` line before two bullets,
		// When it is read, Then both bullets are blocks and the first has
		// its page ref.
		It("does not take the bullets after an unclosed math line into the math", func() {
			root, err := markdown.ParseString("text\n$$\n- a [[P]]\n- b\n", markdown.AsPage())
			Expect(err).ToNot(HaveOccurred())

			for _, node := range root.Children().FilterDeep(content.IsOfType[*content.MathBlock]()) {
				Expect(node.(*content.MathBlock).Value).ToNot(ContainSubstring("- a"))
			}

			blocks := root.Blocks()
			Expect(blocks).To(HaveLen(2))
			Expect(pageRefTitlesOf(blocks[0])).To(Equal([]string{"P"}))
		})

		// Given a page that starts with a byte order mark and a math block,
		// When it is read, Then the blocks after it exist and keep their
		// page ref.
		It("keeps the blocks after a first math block of a page with a byte order mark", func() {
			root, err := markdown.ParseString("\ufeff- $$\n  x^2\n  $$\n- b [[P]]\n- c\n", markdown.AsPage())
			Expect(err).ToNot(HaveOccurred())

			var b, c *content.Block
			for _, block := range root.Blocks() {
				text := ""
				for _, t := range block.Content().FilterDeep(content.IsOfType[*content.Text]()) {
					text += t.(*content.Text).Value
				}
				switch strings.TrimSpace(text) {
				case "b":
					b = block
				case "c":
					c = block
				}
			}
			Expect(b).ToNot(BeNil())
			Expect(c).ToNot(BeNil())
			Expect(pageRefTitlesOf(b)).To(Equal([]string{"P"}))
		})

		// Given a page with a lone `$$` line in a block, When written, Then
		// the text is the same and the block has no math block.
		It("keeps a lone math line as text", func() {
			input := "- x\n  $$\n- b\n"
			Expect(unclosedBlockPageText(input)).To(Equal(input))

			root, err := markdown.ParseString(input, markdown.AsPage())
			Expect(err).ToNot(HaveOccurred())
			Expect(root.Blocks()[0].Children().FilterDeep(content.IsOfType[*content.MathBlock]())).To(BeEmpty())
		})

		// Given a page with a closed math block, When written, Then the
		// text is the same. This works now and must keep working.
		It("keeps a closed math block", func() {
			input := "- $$\n  a\n  $$\n"
			Expect(unclosedBlockPageText(input)).To(Equal(input))
		})
	})

	// Given a lazy `$$` line in a block before two bullets, When written,
	// Then the text is the same.
	FullyEqual("Lazy unclosed math line in a block", "- x\n$$\n- a [[P]]\n- b")

	// Given an HTML block at column 0 before two bullets, When it is read,
	// Then there are three top-level blocks.
	It("does not take the bullets after an HTML line at column 0 into the HTML", func() {
		root, err := markdown.ParseString("- x\n<div>\n- a [[P]]\n- b")
		Expect(err).ToNot(HaveOccurred())
		Expect(root.Blocks()).To(HaveLen(3))
	})

	// Given an unclosed fence in a quote, When written, Then no closing
	// fence is added.
	FullyEqual("Unclosed fence in a quote", "- > ```\n  > x\n- b")

	// Given an unclosed math block, When written, Then no closing line is
	// added.
	FullyEqual("Unclosed math block", "- $$\n  x\n- b")

	// Given a closed math block in a block, When written, Then the text is
	// the same. This works now and must keep working.
	FullyEqual("Closed math block in a block", "- x\n  $$\n  y\n  $$\n- a")

	// Given a closed fence in a quote, When written, Then the text is the
	// same. This works now and must keep working.
	FullyEqual("Closed fence in a quote", "- > ```\n  > x\n  > ```\n- b")
})
