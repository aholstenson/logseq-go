package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// An HTML comment, a `<pre>` or `<script>` element or a `<?php` block can
// span more than one line. Saving the block keeps the last line of the HTML,
// so that the text after it is not read as HTML on the next read.
var _ = Describe("Parsing then outputting HTML blocks over more than one line", func() {
	// Given an HTML comment over three lines in a block, When written,
	// Then the closing line stays.
	FullyEqual("HTML comment over three lines in a block", "- a\n\n  <!--\n  c\n  -->")

	// Given an HTML comment that closes on its second line, When written,
	// Then the second line stays.
	FullyEqual("HTML comment that closes on its second line", "- a\n  <!-- c\n  d -->")

	// Given a pre element over three lines, When written, Then the closing
	// line stays.
	FullyEqual("Pre element over three lines", "- a\n  <pre>\n  x\n  </pre>")

	// Given a script element over three lines, When written, Then the
	// closing line stays.
	FullyEqual("Script element over three lines", "- a\n  <script>\n  x\n  </script>")

	// Given a PHP block over three lines, When written, Then the closing
	// line stays.
	FullyEqual("PHP block over three lines", "- a\n  <?php\n  x\n  ?>")

	// Given an HTML comment on one line, When written, Then the text is the
	// same. This works now and must keep working.
	FullyEqual("HTML comment on one line", "- a\n  <!-- one -->")

	// Given a page that starts with an HTML comment over three lines, When
	// it is read, written and read again, Then the text is the same and the
	// block after the comment is still a block.
	It("keeps the block after an HTML comment at the top of a page", func() {
		input := "<!--\nc\n-->\n- a\n"
		root, err := markdown.ParseString(input, markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())
		v, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())
		if !strings.HasSuffix(v, "\n") {
			v += "\n"
		}
		Expect(v).To(Equal(input))

		again, err := markdown.ParseString(v, markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())
		blocks := again.Blocks()
		Expect(blocks).ToNot(BeEmpty())
		text := blocks[len(blocks)-1].Children().FindDeep(content.IsOfType[*content.Text]())
		Expect(text).ToNot(BeNil())
		Expect(text.(*content.Text).Value).To(Equal("a"))
	})
})
