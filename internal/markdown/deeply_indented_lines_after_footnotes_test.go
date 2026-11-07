package markdown_test

import (
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A footnote definition can be followed by a line that is indented eight or
// more spaces. Reading such a text gives no error, and the text that a save
// writes can be read again.
var _ = Describe("Parsing deeply indented lines after a footnote definition", func() {
	// Given a footnote definition and a line indented ten spaces, When it
	// is read, Then there is no error.
	It("reads a deeply indented line after a footnote definition", func() {
		_, err := markdown.ParseString("[^1]: a\n          - x\n")
		Expect(err).ToNot(HaveOccurred())
	})

	// Given a page with a heading line and a footnote definition with
	// empty items, When it is read, written and read again, Then the
	// second read has no error.
	It("reads the saved text of a footnote definition with deeply indented items", func() {
		root, err := markdown.ParseString("- x\n  ###\n  [^1]:\n    -\n\n\n            -\n", markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())
		v, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())

		_, err = markdown.ParseString(v+"\n", markdown.AsPage())
		Expect(err).ToNot(HaveOccurred(), "text: %q", v)
	})
})
