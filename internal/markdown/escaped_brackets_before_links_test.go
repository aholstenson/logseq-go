package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A `\[` in the file shows link syntax as text, such as `\[x](y)`. `\[` and
// `\]` together start and end display math. Saving the block keeps the text
// and does not add a `\]` that makes it math.
var _ = Describe("Parsing then outputting escaped brackets of link syntax", func() {
	// Given escaped link syntax, When written, Then the text is the same
	// and no math is read from it.
	It("keeps escaped link syntax as text", func() {
		input := "- a \\[x](y) b"
		v := parseAndOutput(input)
		Expect(v).To(Equal(input))

		again, err := markdown.ParseString(v)
		Expect(err).ToNot(HaveOccurred())
		Expect(again.Children().FilterDeep(content.IsOfType[*content.Math]())).To(BeEmpty())
	})

	// Given an escaped page link, When written, Then the text is the same.
	FullyEqual("Escaped page link", "- \\[[x]]")

	// Given a paragraph with the text `[x](y)` made with the API, When
	// written and read again, Then the text is the same and it is not a
	// link or math. This works now and must keep working.
	It("keeps link syntax in a text made with the API as text", func() {
		v, err := markdown.AsString(content.NewParagraph(content.NewText("[x](y)")))
		Expect(err).ToNot(HaveOccurred())

		again, err := markdown.ParseString(v)
		Expect(err).ToNot(HaveOccurred())
		Expect(again.Children().FilterDeep(content.IsOfType[*content.Link]())).To(BeEmpty())
		Expect(again.Children().FilterDeep(content.IsOfType[*content.Math]())).To(BeEmpty())

		text := ""
		for _, node := range again.Children().FilterDeep(content.IsOfType[*content.Text]()) {
			text += node.(*content.Text).Value
		}
		Expect(text).To(Equal("[x](y)"))
	})
})
