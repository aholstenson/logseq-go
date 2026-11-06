package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A saved tag must point at the same page when it is read again. The
// `#[[...]]` form must be kept when the user wrote it, and must be used when
// the bare `#name` form would end the tag early.
var _ = Describe("Tag form in saved blocks", func() {
	pageRefTitles := func(block *content.Block) []string {
		titles := []string{}
		for _, ref := range block.Children().PageReferences() {
			titles = append(titles, ref.(content.PageRef).GetTo())
		}
		return titles
	}

	It("keeps the bracket form of a tag", func() {
		// Given `#[[x]]`, When it is written
		input := "- #[[x]]"

		// Then the file text is unchanged
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("keeps the target of a bracket tag with a comma", func() {
		// Given `#[[x,y]]`
		block, err := markdown.ParseString("- #[[x,y]]")
		Expect(err).ToNot(HaveOccurred())

		// When it is written and read again
		written, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())
		again, err := markdown.ParseString(written)
		Expect(err).ToNot(HaveOccurred())

		// Then the tag target is still `x,y`
		Expect(pageRefTitles(again)).To(Equal([]string{"x,y"}), "written text: %q", written)
	})

	It("writes a tag made with the API so that it reads back to the same title", func() {
		// Given a tag made with the API for a title that ends with a period
		block := content.NewBlock(content.NewParagraph(content.NewHashtag("a.")))

		// When it is written and read again
		written, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())
		again, err := markdown.ParseString(written)
		Expect(err).ToNot(HaveOccurred())

		// Then the tag target is `a.`
		Expect(pageRefTitles(again)).To(Equal([]string{"a."}), "written text: %q", written)
	})

	DescribeTable("writes tags made with the API so that they read back to the same title",
		func(title string, expected string) {
			// Given a tag made with the API
			block := content.NewBlock(content.NewParagraph(content.NewHashtag(title)))

			// When it is written and read again
			written, err := markdown.AsString(block)
			Expect(err).ToNot(HaveOccurred())
			Expect(written).To(Equal(expected))

			again, err := markdown.ParseString(written)
			Expect(err).ToNot(HaveOccurred())

			// Then the tag target is the same title
			Expect(pageRefTitles(again)).To(Equal([]string{title}), "written text: %q", written)
		},
		Entry("a plain title", "x", "#x"),
		Entry("a title with an underscore", "snake_case", "#snake_case"),
		Entry("a title with a period inside", "v1.0", "#v1.0"),
		Entry("a title with a space", "a b", "#[[a b]]"),
		Entry("a title with a comma", "a,b", "#[[a,b]]"),
		Entry("a title with a colon", "a:b", "#[[a:b]]"),
		Entry("a title that ends with an exclamation mark", "a!", "#[[a!]]"),
		Entry("a title with a quote", `a"b`, `#[[a"b]]`),
		Entry("a title with a hash", "a#b", "#[[a#b]]"),
		Entry("a title with a closing bracket", "a]b", "#[[a]b]]"),
	)

	DescribeTable("keeps the form of a tag that is read and written",
		func(input string) {
			Expect(parseAndOutput(input)).To(Equal(input))
		},
		Entry("a bare tag", "- #x"),
		Entry("a bare tag with an underscore", "- #snake_case"),
		Entry("a bracket tag with a space", "- #[[a b]]"),
		Entry("a bracket tag with text directly after it", "- #[[x]]y"),
	)
})
