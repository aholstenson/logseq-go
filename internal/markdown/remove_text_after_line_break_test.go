package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// When the text on the last line of a paragraph is removed, the line break
// before it and the indentation of the removed line must not be written.
var _ = Describe("Removing the text after a line break", func() {
	It("does not leave a line of spaces", func() {
		// Given a block with a paragraph of two lines, and the second line has extra indentation
		root, err := markdown.ParseString("- a\n     b")
		Expect(err).ToNot(HaveOccurred())

		// When the text of the second line is removed and the block is written
		text := root.Children().FindDeep(func(n content.Node) bool {
			t, ok := n.(*content.Text)
			return ok && t.Value == "b"
		})
		Expect(text).ToNot(BeNil())
		text.RemoveSelf()
		output, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())

		// Then only the first line is written
		Expect(output).To(Equal("- a"))
	})
})
