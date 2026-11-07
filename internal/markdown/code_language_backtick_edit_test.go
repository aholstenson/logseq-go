package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The info string of a code fence of backticks cannot have a backtick.
// When a program sets a language with a backtick on such a code block, the
// written code block must still read back as a code block with that
// language.
var _ = Describe("Code language with a backtick", func() {
	It("reads back as a code block with the same language", func() {
		// Given a block with a code fence of backticks
		root, err := markdown.ParseString("- ```js\n  x\n  ```")
		Expect(err).ToNot(HaveOccurred())
		code := root.Children().FindDeep(content.IsOfType[*content.CodeBlock]())
		Expect(code).ToNot(BeNil())

		// When the language is set to a value with a backtick and the block is written and read again
		code.(*content.CodeBlock).WithLanguage("a`b")
		output, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())
		again, err := markdown.ParseString(output)
		Expect(err).ToNot(HaveOccurred())

		// Then it is a code block with the language `a`b`
		readBack := again.Children().FindDeep(content.IsOfType[*content.CodeBlock]())
		Expect(readBack).ToNot(BeNil(), "output: %q", output)
		Expect(readBack.(*content.CodeBlock).Language).To(Equal("a`b"), "output: %q", output)
	})
})
