package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A table at the top of a page can be indented, and the rows of a table
// after a quote line can be lazy lines without the `>` marker. Saving keeps
// these lines as they are in the file.
var _ = Describe("Parsing then outputting the indentation of tables and lazy table rows", func() {
	// Given a page that starts with an indented table, When written, Then
	// the text is the same.
	It("keeps the indentation of a table before the first block", func() {
		input := " | a | b |\n |---|---|\n | 1 | 2 |\n- a\n"
		root, err := markdown.ParseString(input, markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())
		v, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())
		if !strings.HasSuffix(v, "\n") {
			v += "\n"
		}
		Expect(v).To(Equal(input))
	})

	// Given table rows after a quote line without the `>` marker, When
	// written, Then the rows do not get the marker.
	FullyEqual("Table rows after a quote line", "- a\n  > q\n  | x | y |\n  | --- | --- |")
})
