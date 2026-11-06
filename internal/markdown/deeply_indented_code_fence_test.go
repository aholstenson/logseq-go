package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq reads a code fence inside a block as a code block, also when the
// fence is indented more than the text of the block.
var _ = Describe("Parsing a code fence indented more than the text of its block", func() {
	// Given a fence indented with a tab and two spaces under a block, When
	// written, Then the fences stay and the code is not made into an
	// inline code span.
	It("keeps the fences of a code block indented with a tab and two spaces", func() {
		v := parseAndOutput("- a\n\t  ```\n\t  x\n\t  ```\n")

		Expect(strings.Count(v, "```")).To(Equal(2), "written text: %q", v)
		Expect(v).ToNot(ContainSubstring("`x`"))
	})

	// Given a fence indented six spaces under a block, with a code line
	// that is indented two more spaces, When parsed, Then the block has a
	// code block whose code keeps the extra indentation.
	It("reads a code fence indented six spaces as a code block that keeps the code indentation", func() {
		root, err := markdown.ParseString("- a\n      ```\n        y()\n      ```\n")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(1))
		code, ok := root.Blocks()[0].Children().FindDeep(content.IsOfType[*content.CodeBlock]()).(*content.CodeBlock)
		Expect(ok).To(BeTrue(), "the block has a code block")
		Expect(code.Code).To(ContainSubstring("  y()"))
	})

	// Given a fence indented four spaces after a blank line, When parsed,
	// Then it is a fenced code block and not an indented code block that
	// holds the fence lines.
	It("reads a code fence indented four spaces after a blank line as a code block", func() {
		root, err := markdown.ParseString("text\n\n    ```\n    x\n    ```\n")
		Expect(err).ToNot(HaveOccurred())

		code, ok := root.Children().FindDeep(content.IsOfType[*content.CodeBlock]()).(*content.CodeBlock)
		Expect(ok).To(BeTrue(), "the page has a code block")
		Expect(code.Code).To(Equal("x\n"))
	})

	// Given a bullet inside a fence indented six spaces, When parsed, Then
	// the bullet is code and not a child block.
	It("does not read a bullet inside a deeply indented code fence as a block", func() {
		root, err := markdown.ParseString("- #tag\n      ```\n      - not a block\n      ```")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(1))
		Expect(root.Blocks()[0].Blocks()).To(BeEmpty())
	})
})
