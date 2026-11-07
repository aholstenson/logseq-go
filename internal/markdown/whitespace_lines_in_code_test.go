package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Editors often indent the blank lines of code. A line of code in a block
// can then have only spaces. Saving the block keeps the spaces of such a
// line.
var _ = Describe("Parsing then outputting lines of only spaces in code", func() {
	// Given a code block with a line of only spaces, When written, Then
	// the text is the same and the code has the spaces.
	It("keeps a line of only spaces in a code block in a block", func() {
		input := "- ```py\n  a\n      \n  b\n  ```"
		Expect(parseAndOutput(input)).To(Equal(input))

		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		code := root.Children().FindDeep(content.IsOfType[*content.CodeBlock]())
		Expect(code).ToNot(BeNil())
		Expect(code.(*content.CodeBlock).Code).To(Equal("a\n    \nb\n"))
	})
})
