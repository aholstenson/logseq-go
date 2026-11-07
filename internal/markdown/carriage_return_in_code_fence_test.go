package markdown_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A line of a code fence can have a lone carriage return (CR) in it. Logseq
// reads a lone CR as a line break. Saving the block keeps the fence lines,
// so that the code block does not become inline code.
var _ = Describe("Parsing then outputting a lone carriage return in a code fence", func() {
	// Given a code fence with a lone CR in a code line, When written, Then
	// the opening and closing fence lines stay.
	It("keeps the fence lines of a code fence with a lone carriage return", func() {
		v := parseAndOutput("- a\n  ```\n  x\ry\n  ```")

		fences := 0
		for _, line := range strings.Split(v, "\n") {
			if strings.TrimSpace(line) == "```" {
				fences++
			}
		}
		Expect(fences).To(Equal(2), "text: %q", v)
	})
})
