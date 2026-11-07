package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A quote can be only a `>` marker, or start with a blank `>` line. Saving
// the block keeps these lines as they are in the file.
var _ = Describe("Parsing then outputting empty quote lines", func() {
	// Given an empty quote on the bullet line, When written, Then no space
	// is added after the marker.
	FullyEqual("Empty quote", "- >")

	// Given a quote with a blank first line, When written, Then the blank
	// line stays.
	FullyEqual("Blank first quote line", "- >\n  > a")

	// Given an empty quote line under the text of a block, When written
	// two times, Then both outputs are the input.
	It("keeps an empty quote line under the text of a block over two saves", func() {
		input := "- a\n  >"
		first := parseAndOutput(input)
		Expect(first).To(Equal(input))
		Expect(parseAndOutput(first)).To(Equal(input))
	})
})
