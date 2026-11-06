package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq stores a divider as a block that holds only a thematic break, for
// example "- ---".
var _ = Describe("Parsing divider blocks", func() {
	// Given a divider block between two blocks, When parsed, Then there are
	// three blocks and the second one holds the thematic break.
	It("reads a divider block between two blocks as a block of its own", func() {
		root, err := markdown.ParseString("- above\n- ---\n- below")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(3))
		Expect(root.Blocks()[1].Content()).To(HaveLen(1))
		_, ok := root.Blocks()[1].Content()[0].(*content.ThematicBreak)
		Expect(ok).To(BeTrue(), "the second block holds a thematic break")
	})

	// Given a divider block, When parsed and written, Then the text is the
	// same as the input.
	FullyEqual("Divider block between two blocks", "- above\n- ---\n- below")
	FullyEqual("Divider block alone on a page", "- ---")

	// Given a nested divider block, When parsed, Then it is a child block
	// that holds the thematic break.
	It("reads a nested divider block as a child block", func() {
		Expect(emptyParentOutline("- a\n\t- ---\n")).To(Equal([]string{
			"1:a",
			"2:---",
		}))
	})

	// Given a divider block after a child block, When parsed, Then it is the
	// next top-level block.
	It("reads a divider block after a child block as a top-level block", func() {
		Expect(emptyParentOutline("- a\n\t- b\n- ---\n- c\n")).To(Equal([]string{
			"1:a",
			"2:b",
			"1:---",
			"1:c",
		}))
	})

	// Given a divider block with a child, When parsed, Then the child is
	// under the divider block.
	It("keeps the child of a divider block under it", func() {
		Expect(emptyParentOutline("- ---\n\t- c\n")).To(Equal([]string{
			"1:---",
			"2:c",
		}))
	})

	// Given a thematic break in the text before the first bullet, When
	// parsed, Then it is not a block.
	It("keeps a thematic break without a bullet in the page text", func() {
		root, err := markdown.ParseString("a\n\n---\n\n- b\n")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(1))
		Expect(root.Content()).To(HaveLen(2))
		_, ok := root.Content()[1].(*content.ThematicBreak)
		Expect(ok).To(BeTrue(), "the page text holds a thematic break")
	})

	FullyEqual("Nested divider block", "- a\n\t- ---")
	FullyEqual("Divider block after a child block", "- a\n\t- b\n- ---\n- c")

	// Given a divider block written with "***", When it is written, read
	// and written again, Then both written texts are the same.
	It("writes a divider block made with asterisks the same way each time", func() {
		first := parseAndOutput("- ***")
		second := parseAndOutput(first)
		Expect(second).To(Equal(first))
	})
})
