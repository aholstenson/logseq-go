package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Drawers and HTML elements are text in Logseq, so they are written back as
// they are.
var _ = Describe("Drawers and HTML elements in saved blocks", func() {
	DescribeTable("writes the text unchanged",
		func(input string) {
			Expect(parseAndOutput(input)).To(Equal(input))
		},
		Entry("a drawer on the line of the bullet", "- :notes:\n  [[x]]\n  :end:"),
		Entry("a drawer after text, with a child block", "- a\n  text\n  :notes:\n  [[x]]\n  :END:\n\t- child"),
		Entry("a drawer with blank lines and an end line with spaces", "- a\n  :notes:\n  x\n\n  y\n   :End: \n  after"),
		Entry("an empty drawer", "- a\n  :notes:\n  :end:"),
		Entry("a drawer with Windows line endings", "- a\r\n  :notes:\r\n  [[x]]\r\n  :END:\r\n  [[y]]"),
		Entry("an HTML element", `- x <span class="y">[[a]] *b*</span> z`),
		Entry("an HTML element on two lines", "- a\n  x <span>\n  [[a]]</span> [[b]]"),
	)

	It("writes a drawer made with the API", func() {
		// Given a block with a drawer made with the API
		block := content.NewBlock(content.NewParagraph(content.NewText("a")), content.NewDrawer("notes", "x\n"))

		// When it is written in a page
		output, err := markdown.AsString(content.NewBlock(block))
		Expect(err).ToNot(HaveOccurred())

		// Then it has the name, the content and an :END: line
		Expect(output).To(Equal("- a\n  :notes:\n  x\n  :END:"))
	})

	It("writes an end line that does not end a drawer as :END:", func() {
		// Given a drawer with an end line that is not an end line
		block := content.NewBlock(content.NewParagraph(content.NewText("a")), content.NewDrawer("notes", "x\n").WithEndLine(":done:"))

		// When it is written in a page
		output, err := markdown.AsString(content.NewBlock(block))
		Expect(err).ToNot(HaveOccurred())

		// Then the drawer ends with :END:
		Expect(output).To(Equal("- a\n  :notes:\n  x\n  :END:"))
	})
})
