package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mathBlockValue parses the input and returns the value of the first math
// block in it.
func mathBlockValue(input string) string {
	block, err := markdown.ParseString(input)
	Expect(err).ToNot(HaveOccurred())

	found := block.Children().FindDeep(func(node content.Node) bool {
		_, ok := node.(*content.MathBlock)
		return ok
	})
	if found == nil {
		return ""
	}

	return found.(*content.MathBlock).Value
}

// The lines inside a math block, a drawer, a logbook or a begin block lose
// the indentation of the first line of the block. A line with less
// indentation than that keeps its text from the first character that is not a
// space.
var _ = Describe("Parsing block lines with less indentation than the block", func() {
	// Given a math block in a block, and a line in it that is indented with a
	// tab, When parsed, Then the line is in the math block and nothing
	// panics.
	It("keeps a tab indented line of a math block", func() {
		input := "- ```\n   $$\n\t-\n   $$\n  - a\n"
		Expect(func() { mathBlockValue(input) }).ToNot(Panic())
		Expect(mathBlockValue(input)).To(HavePrefix("-\n"))
	})

	// Given a math block after a quote, and a line with a tab and `>` as its
	// last line, When parsed, Then the line is in the math block and nothing
	// panics.
	It("keeps a tab indented last line of a math block", func() {
		input := "- > q\n   $$\n\t>\n   $$"
		Expect(func() { mathBlockValue(input) }).ToNot(Panic())
		Expect(mathBlockValue(input)).To(Equal(">\n"))
	})
})
