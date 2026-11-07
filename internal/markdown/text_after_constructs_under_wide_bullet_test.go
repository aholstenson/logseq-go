package markdown_test

import (
	"regexp"
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// textAfterConstructPageText reads a page and writes it as a page is
// saved: the writer output with a line break at the end.
func textAfterConstructPageText(input string) string {
	root, err := markdown.ParseString(input, markdown.AsPage())
	Expect(err).ToNot(HaveOccurred())
	v, err := markdown.AsString(root)
	Expect(err).ToNot(HaveOccurred())
	if v != "" && !strings.HasSuffix(v, "\n") {
		v += "\n"
	}
	return v
}

// hasTextParagraphAfterProperties tells if the first block of a page has a
// paragraph with the text `text` after its properties.
func hasTextParagraphAfterProperties(page string, text string) bool {
	root, err := markdown.ParseString(page, markdown.AsPage())
	Expect(err).ToNot(HaveOccurred())
	blocks := root.Blocks()
	if len(blocks) == 0 {
		return false
	}

	afterProperties := false
	for _, node := range blocks[0].Children() {
		switch n := node.(type) {
		case *content.Properties:
			afterProperties = true
		case *content.Paragraph:
			value := ""
			for _, t := range n.Children().FilterDeep(content.IsOfType[*content.Text]()) {
				value += t.(*content.Text).Value
			}
			if afterProperties && value == text {
				return true
			}
		}
	}
	return false
}

// A bullet can have more than one space or a tab after the dash (`-   t`).
// A text line after a code fence, quote or property line in such a block is
// at the text column of the bullet. Saving the block keeps the line at that
// column, so that every later save gives the same text.
var _ = Describe("Parsing then outputting text after a construct under a bullet with more than one space", func() {
	// Given a text line after a code fence under a three-space bullet,
	// When written three times, Then every output is the input and the
	// line `c` never has more than four spaces before it.
	It("keeps a text line after a code fence over three saves", func() {
		input := "-   t\n    ```\n    x\n    ```\n    c"
		wide := regexp.MustCompile(`(?m)^ {5,}c$`)
		v := input
		for i := 0; i < 3; i++ {
			v = parseAndOutput(v)
			Expect(wide.MatchString(v)).To(BeFalse(), "save %d: %q", i+1, v)
			Expect(v).To(Equal(input), "save %d", i+1)
		}
	})

	// Given a page with a text line after a property line under a
	// three-space bullet, When written three times, Then the third output
	// is the first output and each output has the text after the
	// properties.
	It("keeps a text line after a property line over three saves", func() {
		outputs := []string{}
		v := "-   t\n    a:: b\n    c\n"
		for i := 0; i < 3; i++ {
			v = textAfterConstructPageText(v)
			outputs = append(outputs, v)
			Expect(hasTextParagraphAfterProperties(v, "c")).To(BeTrue(), "save %d: %q", i+1, v)
		}
		Expect(outputs[2]).To(Equal(outputs[0]))
	})

	// Given a page with a text line after a quote under a tab bullet, When
	// written two times, Then the second output is the first output.
	It("keeps a text line after a quote under a tab bullet over two saves", func() {
		first := textAfterConstructPageText("-\tItem\n\n\t> quote\n\n\tMore text\n")
		Expect(textAfterConstructPageText(first)).To(Equal(first))
	})

	// Given a page with a child block with a three-space bullet and a text
	// line after a code fence, When written two times, Then the second
	// output is the first output.
	It("keeps a text line after a code fence in a child block over two saves", func() {
		first := textAfterConstructPageText("- p\n\t-   t\n\t    ```\n\t    x\n\t    ```\n\n\t    c\n")
		Expect(textAfterConstructPageText(first)).To(Equal(first))
	})

	// Given a page with a text line after a code fence under a one-space
	// bullet, When written two times, Then both outputs are the input. This
	// works now and must keep working.
	It("keeps a text line after a code fence under a one-space bullet", func() {
		input := "- t\n  ```\n  x\n  ```\n    c\n"
		first := textAfterConstructPageText(input)
		Expect(first).To(Equal(input))
		Expect(textAfterConstructPageText(first)).To(Equal(input))
	})
})
