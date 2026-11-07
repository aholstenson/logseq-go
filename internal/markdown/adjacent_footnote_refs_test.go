package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A footnote ref can be right before a `[`, such as a second footnote ref
// (`x[^1][^2]`) or a link (`[^1][l](u)`). Logseq reads both as footnote
// refs. Saving the block keeps the footnote ref and does not escape it.
var _ = Describe("Parsing then outputting footnote refs before a bracket", func() {
	// Given two footnote refs next to each other, When written, Then the
	// text is the same and both are read as footnote refs.
	It("keeps two footnote refs next to each other", func() {
		input := "- x[^1][^2]"
		Expect(parseAndOutput(input)).To(Equal(input))

		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(root.Children().FilterDeep(content.IsOfType[*content.FootnoteRef]())).To(HaveLen(2))
	})

	// Given a footnote ref right before a link, When written, Then the
	// footnote ref is not escaped.
	FullyEqual("Footnote ref before a link", "- x[^1][l](u)\n\n  [^1]: d")

	// Given two footnote refs with a space between them, When written, Then
	// the text is the same. This works now and must keep working.
	FullyEqual("Footnote refs with a space between them", "- x[^1] [^2]")
})
