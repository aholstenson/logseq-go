package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// wideBulletPageText reads a page and writes it as a page is saved: the
// writer output with a line break at the end.
func wideBulletPageText(input string) string {
	root, err := markdown.ParseString(input, markdown.AsPage())
	Expect(err).ToNot(HaveOccurred())
	v, err := markdown.AsString(root)
	Expect(err).ToNot(HaveOccurred())
	if v != "" && !strings.HasSuffix(v, "\n") {
		v += "\n"
	}
	return v
}

// Some tools write a bullet with more than one space or a tab after the
// dash (`-   x`, `-\tx`). The text of the block then starts at a column
// after two. Saving such a block keeps a code fence, quote, list, math
// block, table or SCHEDULED line in the block at the column of its text.
var _ = Describe("Parsing then outputting content under a bullet with more than one space", func() {
	// Given a code fence under a bullet with three spaces, When written,
	// Then the text is the same and it reads back as a code block.
	It("keeps a code fence under a bullet with three spaces", func() {
		input := "-   ```\n    x\n    ```"
		v := parseAndOutput(input)
		Expect(v).To(Equal(input))

		again, err := markdown.ParseString(v)
		Expect(err).ToNot(HaveOccurred())
		code := again.Children().FindDeep(content.IsOfType[*content.CodeBlock]())
		Expect(code).ToNot(BeNil())
		Expect(code.(*content.CodeBlock).Code).To(Equal("x\n"))
	})

	// Given a code fence under a tab bullet, When written, read and written
	// again, Then both outputs are the input and no code span is read.
	It("keeps a code fence under a tab bullet over two saves", func() {
		input := "-\t```js\n\tx\n\t```"
		first := parseAndOutput(input)
		Expect(first).To(Equal(input))

		second := parseAndOutput(first)
		Expect(second).To(Equal(input))

		again, err := markdown.ParseString(second)
		Expect(err).ToNot(HaveOccurred())
		Expect(again.Children().FilterDeep(content.IsOfType[*content.CodeSpan]())).To(BeEmpty())
	})

	// Given a quote under a bullet with three spaces, When written, Then
	// the text is the same and it reads back as one quote.
	It("keeps a quote under a bullet with three spaces", func() {
		input := "-   > a\n    > b"
		v := parseAndOutput(input)
		Expect(v).To(Equal(input))

		again, err := markdown.ParseString(v)
		Expect(err).ToNot(HaveOccurred())
		Expect(again.Children().FilterDeep(content.IsOfType[*content.Blockquote]())).To(HaveLen(1))
	})

	// Given a list under a bullet with three spaces, When written, Then the
	// text is the same.
	FullyEqual("List under a bullet with three spaces", "-   * a\n    * b")

	// Given a math block under a bullet with three spaces, When written,
	// Then the text is the same.
	FullyEqual("Math block under a bullet with three spaces", "-   $$\n    x\n    $$")

	// Given a table under a bullet with three spaces, When written two
	// times, Then both outputs are the input.
	It("keeps a table under a bullet with three spaces over two saves", func() {
		input := "-   x\n    | a |\n    |---|\n    | 1 |"
		first := parseAndOutput(input)
		Expect(first).To(Equal(input))
		Expect(parseAndOutput(first)).To(Equal(input))
	})

	// Given a page with a SCHEDULED line under a bullet with three spaces,
	// When written two times, Then both outputs are the input.
	It("keeps a SCHEDULED line under a bullet with three spaces over two saves", func() {
		input := "-   TODO x\n    SCHEDULED: <2024-01-01 Mon>\n"
		first := wideBulletPageText(input)
		Expect(first).To(Equal(input))
		Expect(wideBulletPageText(first)).To(Equal(input))
	})

	// Given a page with a table on the line of a bullet with two spaces,
	// When written three times, Then every output is the input.
	It("keeps a table on the line of a bullet with two spaces over three saves", func() {
		input := "-  | a |\n   |---|\n   | 1 |\n"
		v := input
		for i := 0; i < 3; i++ {
			v = wideBulletPageText(v)
			Expect(v).To(Equal(input), "save %d", i+1)
		}
	})

	// Given a paragraph under a bullet with three spaces, When written,
	// Then the text is the same. This works now and must keep working.
	FullyEqual("Paragraph under a bullet with three spaces", "-   a\n    b")

	// Given a page with a table under a bullet with one space, When
	// written, Then the text is the same. This works now and must keep
	// working.
	It("keeps a table under a bullet with one space", func() {
		input := "- x\n  | a |\n  |---|\n  | 1 |\n"
		Expect(wideBulletPageText(input)).To(Equal(input))
	})
})
