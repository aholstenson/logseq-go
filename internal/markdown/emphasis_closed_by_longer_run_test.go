package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Emphasis can be closed by a run of two markers after a space, as in
// `*a **`. Logseq reads it as italic text. Saving the block keeps the
// markers, and a second save gives the same text.
var _ = Describe("Parsing then outputting emphasis closed by a longer marker run", func() {
	// Given emphasis closed by a run of two markers at the end of the
	// line, When written two times, Then both outputs are the input.
	It("keeps emphasis closed by a run of two markers over two saves", func() {
		input := "- *a **"
		first := parseAndOutput(input)
		Expect(first).To(Equal(input))
		Expect(parseAndOutput(first)).To(Equal(input))
	})

	// Given normal emphasis, When written, Then the text is the same. This
	// works now and must keep working.
	FullyEqual("Normal emphasis", "- *a* b")
})
