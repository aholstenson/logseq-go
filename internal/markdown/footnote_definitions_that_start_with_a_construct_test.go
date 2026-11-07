package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The first content of a footnote definition can be a heading or a list on
// the line of the label, as in "[^1]: ## h". Saving the definition gives
// the same text on each save.
var _ = Describe("Parsing then outputting footnote definitions that start with a construct", func() {
	// Given a heading on the label line, When written two times, Then both
	// outputs are the input.
	It("keeps a heading on the label line over two saves", func() {
		input := "x[^1]\n\n[^1]: ## h\n    b"
		first := parseAndOutput(input)
		Expect(first).To(Equal(input))
		Expect(parseAndOutput(first)).To(Equal(input))
	})

	// Given a list on the label line, When written four times, Then the
	// fourth output is the first output.
	It("keeps a list on the label line over four saves", func() {
		outputs := []string{}
		v := "x[^1]\n\n[^1]: * a\n      b"
		for i := 0; i < 4; i++ {
			v = parseAndOutput(v)
			outputs = append(outputs, v)
		}
		Expect(outputs[3]).To(Equal(outputs[0]))
	})

	// Given a paragraph on the label line, When written, Then the text is
	// the same. This works now and must keep working.
	FullyEqual("Paragraph on the label line", "x[^1]\n\n[^1]: a\n    b")
})
