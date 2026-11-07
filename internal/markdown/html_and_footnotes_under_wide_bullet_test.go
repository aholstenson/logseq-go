package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// wideHTMLPageText reads a page and writes it as a page is saved: the
// writer output with a line break at the end.
func wideHTMLPageText(input string) string {
	root, err := markdown.ParseString(input, markdown.AsPage())
	Expect(err).ToNot(HaveOccurred())
	v, err := markdown.AsString(root)
	Expect(err).ToNot(HaveOccurred())
	if v != "" && !strings.HasSuffix(v, "\n") {
		v += "\n"
	}
	return v
}

// A bullet can have more than one space after the dash. An HTML block or a
// footnote definition after the first line of such a block is at the text
// column of the bullet. Saving the block gives the same text on each save.
var _ = Describe("Parsing then outputting HTML and footnotes after the first line under a bullet with more than one space", func() {
	// Given an HTML block after the first line under a three-space bullet,
	// When written two times, Then both outputs are the input.
	It("keeps an HTML block under a three-space bullet over two saves", func() {
		input := "-   t\n    <div>\n    x\n    </div>"
		first := parseAndOutput(input)
		Expect(first).To(Equal(input))
		Expect(parseAndOutput(first)).To(Equal(input))
	})

	// Given a page with a footnote definition after the first line under a
	// three-space bullet, When written three times, Then the third output
	// is the first output.
	It("keeps a footnote definition under a three-space bullet over three saves", func() {
		first := wideHTMLPageText("-   t\n    [^1]: a\n        b\n")
		second := wideHTMLPageText(first)
		third := wideHTMLPageText(second)
		Expect(third).To(Equal(first))
	})
})
