package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A backtick run at the end of a line can start a code span that ends on a
// later line, as in "- a ```\n  code\n  ```". Saving the block keeps the
// lines and the backticks as they are in the file.
var _ = Describe("Parsing then outputting code spans over more than one line", func() {
	// Given a backtick run at the end of the bullet line, When written,
	// Then the three lines stay.
	FullyEqual("Backtick run at the end of the bullet line", "- a ```\n  code\n  ```")

	// Given a code span with two backticks over two lines, When written,
	// Then the backticks stay.
	FullyEqual("Two-backtick code span over two lines", "- a ``x\n  y``")

	// Given a page that starts with a byte order mark and a fence, When
	// written, Then the text is the same.
	It("keeps a fence after a byte order mark at the start of a page", func() {
		input := "\ufeff```\ncode\n```\n- a\n"
		root, err := markdown.ParseString(input, markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())
		v, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())
		if !strings.HasSuffix(v, "\n") {
			v += "\n"
		}
		Expect(v).To(Equal(input))
	})

	// Given a code span on one line, When written, Then the text is the
	// same. This works now and must keep working.
	FullyEqual("Code span on one line", "- a `x` b")
})
