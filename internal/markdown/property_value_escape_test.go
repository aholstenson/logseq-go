package markdown_test

import (
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq reads the value of a property as raw text: it does not remove
// backslashes from it. The writer must not add Markdown escapes to a
// property value, as Logseq would show them as part of the value.
var _ = Describe("Markdown characters in property values", func() {
	It("keeps a star in a block property value without a backslash", func() {
		// Given a block with the property value `2*3`
		input := "- a\n  p:: 2*3"

		// When it is read and written, Then the text is the same
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("keeps an underscore at the end of a page property value without a backslash", func() {
		// Given a page with the page property value `a_b_`
		input := "p:: a_b_\n\n- a\n"

		// When the page is read and written
		root, err := markdown.ParseString(input, markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())
		output, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())

		// Then the text is the same (the writer does not add the last line break)
		Expect(output + "\n").To(Equal(input))
	})
})
