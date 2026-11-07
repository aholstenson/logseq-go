package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// emptyBlockPageText reads a page and writes it as a page is saved: the
// writer output with a line break at the end.
func emptyBlockPageText(input string) string {
	root, err := markdown.ParseString(input, markdown.AsPage())
	Expect(err).ToNot(HaveOccurred())
	v, err := markdown.AsString(root)
	Expect(err).ToNot(HaveOccurred())
	if v != "" && !strings.HasSuffix(v, "\n") {
		v += "\n"
	}
	return v
}

// The code block command of Logseq adds an empty fence ("```\n```"). Saving
// a page keeps an empty code block or math block empty, without a blank
// line between the opening and the closing line.
var _ = Describe("Parsing then outputting empty code and math blocks", func() {
	// Given an empty code block in a block, When written, Then no blank
	// line is added.
	FullyEqual("Empty code block in a block", "- ```\n  ```")

	// Given an empty math block in a block, When written, Then no blank
	// line is added.
	FullyEqual("Empty math block in a block", "- $$\n  $$")

	// Given an empty code block with a language at the top, When written,
	// Then no blank line is added.
	FullyEqual("Empty code block with a language at the top", "```js\n```")

	// Given a page with an empty math block, When written, Then the page
	// text is the same.
	It("keeps an empty math block of a page", func() {
		input := "- $$\n  $$\n"
		Expect(emptyBlockPageText(input)).To(Equal(input))
	})

	// Given a code block with one empty line, When written, Then the empty
	// line stays. This works now and must keep working.
	FullyEqual("Code block with one empty line", "- ```\n\n  ```")
})
