package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The first line of a block can be only the bullet, with a blank line after
// it. Saving the page keeps the blank line.
var _ = Describe("Parsing then outputting a blank line after an empty bullet line", func() {
	// Given a page with an empty bullet line, a blank line and text, When
	// written, Then the blank line stays.
	It("keeps a blank line after an empty bullet line", func() {
		input := "-\n\n  text\n"
		root, err := markdown.ParseString(input, markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())
		v, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())
		if !strings.HasSuffix(v, "\n") {
			v += "\n"
		}
		Expect(v).To(Equal(input))
	})
})
