package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq has no backslash hard line break: it shows a `\` at the end of a
// line as text. So a line that ends with spaces must be saved with the same
// spaces, and not as `\` + newline.
var _ = Describe("Line ends in saved blocks", func() {
	It("keeps a two-space hard line break as two spaces", func() {
		// Given a block line that ends with two spaces
		input := "- line one  \n  hard break"

		// When it is parsed and written
		written := parseAndOutput(input)

		// Then the file text is unchanged and has no backslash
		Expect(written).ToNot(ContainSubstring(`\`))
		Expect(written).To(Equal(input))
	})

	It("keeps a two-space hard line break in a short block", func() {
		// Given `- a  \n  b`, When it is written, Then it is unchanged
		input := "- a  \n  b"
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("keeps one trailing space at the end of a line", func() {
		// Given a block line that ends with one space, When it is written
		input := "- a \n  b"

		// Then the trailing space is kept
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("keeps a backslash hard line break as a backslash", func() {
		// Given a block line that ends with a backslash, When it is written
		input := "- a\\\n  b"

		// Then the file text is unchanged
		Expect(parseAndOutput(input)).To(Equal(input))
	})
})
