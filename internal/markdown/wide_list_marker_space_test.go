package markdown_test

import (
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// More than four columns of space after the marker of a list item, as in
// "*\t\t-", start the text of the item with indentation of its own. Such a
// line must be read without an error, and written back as it is, so that its
// text does not become a new list or a block.
var _ = Describe("A list item with more than four columns of space after the marker", func() {
	DescribeTable("is read and written back as it is",
		func(input string) {
			// Given the text
			// When it is read
			root, err := markdown.ParseString(input)

			// Then there is no error
			Expect(err).ToNot(HaveOccurred())

			// And it is written back as it is, also after it is read again
			output, err := markdown.AsString(root)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal(input))

			again, err := markdown.ParseString(output)
			Expect(err).ToNot(HaveOccurred())
			Expect(markdown.AsString(again)).To(Equal(input))
		},
		Entry("a dash after two tabs", "*\t\t-"),
		Entry("a dash and text after two tabs", "*\t\t- a"),
		Entry("in a block", "- *\t\t-"),
		Entry("in a child list of a block", "- a\n  *\t\t- b"),
		Entry("in a quote", "- > *\t\t- a"),
		Entry("after an ordered marker", "1.\t\t-"),
		Entry("after a plus marker", "+\t\t-"),
		Entry("with a second line", "- *      a\n    b"),
	)

	It("reads a quote with two tabs before a dash", func() {
		// Given a quote marker, two tabs and a dash
		// When it is read and written twice
		root, err := markdown.ParseString(">\t\t-")
		Expect(err).ToNot(HaveOccurred())
		output, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())
		again, err := markdown.ParseString(output)
		Expect(err).ToNot(HaveOccurred())

		// Then the second output is the same as the first
		Expect(markdown.AsString(again)).To(Equal(output))
	})
})
