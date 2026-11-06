package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	"github.com/aholstenson/logseq-go/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A page can use Windows line endings (CRLF), and Logseq also reads a lone
// carriage return (CR) as a line break.
var _ = Describe("Parsing then outputting pages with other line endings", func() {
	// Given blocks with CRLF line endings, When parsed and written, Then
	// the line endings stay CRLF.
	FullyEqual("Blocks with CRLF line endings", "- a\r\n\t- b")

	// Given a block with a code block and CRLF line endings, When written,
	// Then the output does not mix CRLF and LF line endings.
	It("does not mix line endings when a block with a code block and CRLF line endings is written", func() {
		v := parseAndOutput("- a\r\n  ```\r\n  code\r\n  ```\r\n")

		hasCRLF := strings.Contains(v, "\r\n")
		hasBareLF := strings.Contains(strings.ReplaceAll(v, "\r\n", ""), "\n")
		Expect(hasCRLF && hasBareLF).To(BeFalse(), "written text: %q", v)
	})

	// Given two bullets on lines that end with a lone CR, When parsed, Then
	// there are two blocks.
	It("reads bullets on lines that end with a lone carriage return as separate blocks", func() {
		root, err := markdown.ParseString("- a\r- b\r")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(2))
	})

	FullyEqual("Blocks with lone carriage return line endings", "- a\r- b")
	FullyEqual("Task with a date and a logbook with CRLF line endings", "- TODO a [[p]]\r\n  SCHEDULED: <2024-01-01 Mon>\r\n  :LOGBOOK:\r\n  CLOCK: [2024-01-01 Mon 10:00:00]\r\n  :END:\r\n\t- b")
	FullyEqual("Front matter with CRLF line endings", "---\r\ntitle: x\r\n---\r\n- a")
	FullyEqual("Table with CRLF line endings", "- | a | b |\r\n  |---|---|\r\n  | 1 | 2 |")
	FullyEqual("Hard line break with CRLF line endings", "- a  \r\n  b")

	DescribeTable("reads text with other line endings the same as with LF line endings",
		func(input string) {
			// Given a text with CRLF or lone CR line endings
			// When it is parsed
			root, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			// Then it is the same as the text with LF line endings
			lf, err := markdown.ParseString(strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(input))
			Expect(err).ToNot(HaveOccurred())
			Expect(root).To(tests.EqualNode(lf))
		},
		Entry("a task with properties", "title:: x\r\n\r\n- TODO a\r\n  b:: c\r\n"),
		Entry("a code block", "- a\r\n  ```\r\n  code\r\n  ```\r\n"),
		Entry("a quote command", "- #+BEGIN_QUOTE\r\n  x [[p]]\r\n  #+END_QUOTE\r\n"),
		Entry("bullets with lone carriage returns", "- a\r- b\r\t- c\r"),
	)

	It("writes a text with mixed line endings with the most common one", func() {
		// Given a text with two CRLF line endings and one LF line ending
		// When it is parsed and written
		v := parseAndOutput("- a\r\n- b\n- c\r\n")

		// Then every line ends with CRLF
		Expect(v).To(Equal("- a\r\n- b\r\n- c"))
	})

	It("writes blocks made with the API with LF line endings", func() {
		// Given a block made with the API
		block := content.NewBlock(content.NewBlock(content.NewParagraph(content.NewText("a"))), content.NewBlock(content.NewParagraph(content.NewText("b"))))

		// When it is written
		v, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())

		// Then it has LF line endings
		Expect(v).To(Equal("- a\n- b"))
	})
})
