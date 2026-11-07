package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A saved footnote definition must keep the indentation of its lines as the
// user wrote them.
var _ = Describe("Footnote definitions in saved blocks", func() {
	It("keeps the indentation of a lazy continuation line", func() {
		// Given a footnote definition with a continuation line at block indentation
		input := "- x\n  [^1]: def\n  continued [[fn3]]"

		// When it is parsed and written
		// Then the file text is unchanged
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	DescribeTable("keeps the lines of a footnote definition as written",
		func(input string) {
			Expect(parseAndOutput(input)).To(Equal(input))
		},
		Entry("lazy line outside a block", "[^1]: def\ncontinued"),
		Entry("indented line", "- x\n  [^1]: def\n      continued"),
		Entry("lazy line and then an indented line", "- x\n  [^1]: def\n  lazy\n      indented"),
		Entry("lazy line and then a block", "- x\n  [^1]: def\n  continued\n- y\n  z"),
		Entry("lazy line in a child block", "- x\n\t- y\n\t  [^1]: def\n\t  continued\n\t- z"),
	)
})
