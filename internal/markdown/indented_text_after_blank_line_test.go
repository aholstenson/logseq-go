package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq has no indented code blocks: a line indented four or more spaces
// past the text of its block, after a blank line, is plain text.
var _ = Describe("Parsing indented text after a blank line in a block", func() {
	// Given an indented page link after a blank line, When parsed, Then the
	// block references the page.
	It("reads a page link in indented text after a blank line as a reference", func() {
		root, err := markdown.ParseString("- x\n\n      [[a]]")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(1))
		refs := root.Blocks()[0].Content().PageReferences()
		Expect(refs).To(HaveLen(1))
		Expect(refs[0].(content.PageRef).GetTo()).To(Equal("a"))
	})

	// Given indented text after a blank line, When parsed, Then the block
	// has no code block.
	It("does not read indented text after a blank line as a code block", func() {
		root, err := markdown.ParseString("- a\n\n      code")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Children().FindDeep(content.IsOfType[*content.CodeBlock]())).To(BeNil())
	})

	// Given indented text after a blank line, When written, Then the text
	// is the same and no code fence is added.
	FullyEqual("Indented page link after a blank line", "- x\n\n      [[a]]")
	FullyEqual("Indented text after a blank line", "- a\n\n      code")

	// Given indented text that starts with `# ` after a blank line, When
	// written, Then no backslash is added, as an indented line is not a
	// heading.
	FullyEqual("Indented text that starts with a number sign after a blank line", "- x\n\n      # h")

	// Given indented text with more lines, When written, Then each line
	// keeps its indentation.
	FullyEqual("Indented text with more lines after a blank line", "- a\n\n      line 1\n      line 2")
})
