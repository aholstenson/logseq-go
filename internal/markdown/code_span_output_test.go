package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A saved code span must keep the backtick markers and padding that the user
// wrote. Code spans are read with the rules of CommonMark, which differ from
// Logseq for padding and for backticks inside the code.
var _ = Describe("Code spans in saved text", func() {
	It("keeps the spaces around the code in a padded code span", func() {
		// Given a code span with a space on each side
		input := "- ` a `"

		// When it is parsed
		block, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())

		// Then the code is read with the rules of CommonMark, which remove
		// one space at each end
		code := block.Children().FindDeep(content.IsOfType[*content.CodeSpan]())
		Expect(code).ToNot(BeNil())
		Expect(code.(*content.CodeSpan).Value).To(Equal("a"))

		// And when it is written, Then the file text is unchanged
		written, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())
		Expect(written).To(Equal(input))
	})

	It("keeps the markers of a code span that holds double backticks", func() {
		// Given a code span with double backticks inside, When it is written
		input := "- `a ``b`` c`"

		// Then the file text is unchanged
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	DescribeTable("keeps the text of a code span",
		func(input string) {
			Expect(parseAndOutput(input)).To(Equal(input))
		},
		Entry("with double backticks", "- ``a``"),
		Entry("with a backtick inside", "- ``x ` y``"),
		Entry("with two spaces on each side", "- `  a  `"),
		Entry("with only spaces", "- `  `"),
		Entry("next to other text", "- see ` a ` and ``b``."),
	)

	It("writes a changed code span with markers that keep the code", func() {
		// Given a code span read from a file
		block, err := markdown.ParseString("- ` a `")
		Expect(err).ToNot(HaveOccurred())

		// When its code is changed to hold a backtick
		code := block.Children().FindDeep(content.IsOfType[*content.CodeSpan]()).(*content.CodeSpan)
		code.Value = "a ` b"

		// Then it is written with markers that read back as the new code
		written, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())
		Expect(written).To(Equal("- ``a ` b``"))
	})
})
