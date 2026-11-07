package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Some text looks like markup but the reader does not read it as markup,
// such as `{{ name }}` with spaces or `*` alone in a heading. Saving the
// block does not add a backslash before such text.
var _ = Describe("Parsing then outputting text that needs no backslash", func() {
	// Given macro-like text with spaces in the braces, When written, Then
	// no backslash is added.
	FullyEqual("Macro-like text with spaces", "- use {{ name }} here")

	// Given a star in a heading, When written, Then no backslash is added.
	FullyEqual("Star in a heading", "- ## 2*3")

	// Given a macro, When written, Then the text is the same and it is read
	// as a macro. This works now and must keep working.
	It("keeps a macro", func() {
		input := "- {{m a, b}}"
		v := parseAndOutput(input)
		Expect(v).To(Equal(input))

		again, err := markdown.ParseString(v)
		Expect(err).ToNot(HaveOccurred())
		Expect(again.Children().FilterDeep(content.IsOfType[*content.Macro]())).To(HaveLen(1))
	})
})
