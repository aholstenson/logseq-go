package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// unclosedFencePageTitles gives the titles of the pages that the content of
// the block references.
func unclosedFencePageTitles(b *content.Block) []string {
	titles := []string{}
	for _, ref := range b.Content().PageReferences() {
		titles = append(titles, ref.(content.PageRef).GetTo())
	}
	return titles
}

// Logseq reads a code fence or a "#+BEGIN_" line that is not closed as text.
// The lines after it are text, page links and child blocks as usual, and
// saving the page does not add a closing line.
var _ = Describe("Parsing a code fence or a begin line that is not closed", func() {
	// Given a block with a code fence that is not closed, When written,
	// Then the text is the same and no closing fence is added.
	FullyEqual("Code fence that is not closed", "- a\n  ```\n  unclosed [[p]]")

	// Given the same block, When parsed, Then the page link after the
	// fence is a reference.
	It("reads a page link after a code fence that is not closed as a reference", func() {
		root, err := markdown.ParseString("- a\n  ```\n  unclosed [[p]]")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(1))
		Expect(unclosedFencePageTitles(root.Blocks()[0])).To(ContainElement("p"))
	})

	// Given a block with a quote begin line that is not closed, When
	// parsed, Then the page link after it is a reference.
	It("reads a page link after a quote begin line that is not closed as a reference", func() {
		root, err := markdown.ParseString("- n\n  #+BEGIN_QUOTE\n  never closed [[p]]")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(1))
		Expect(unclosedFencePageTitles(root.Blocks()[0])).To(ContainElement("p"))
	})

	// Given a block with a code fence that is not closed and a child bullet
	// after it, When parsed, Then the bullet is a child block.
	It("reads a bullet after a code fence that is not closed as a child block", func() {
		root, err := markdown.ParseString("- a\n  ```\n  - b\n- c")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(2))
		a := root.Blocks()[0]
		Expect(a.Blocks()).To(HaveLen(1))
		Expect(markdown.AsString(a.Blocks()[0].Content()[0])).To(Equal("b"))
	})

	// Given the same page, When written, Then no second fence line is
	// added.
	It("does not add a closing fence after a code fence that is not closed", func() {
		v := parseAndOutput("- a\n  ```\n  - b\n- c")
		Expect(strings.Count(v, "```")).To(Equal(1), "written text: %q", v)
	})

	// Given a code fence that only a fence line in the next block closes,
	// When parsed, Then neither block has a code block, and When written,
	// Then the text is the same.
	It("does not read a code fence that only the next block closes as code", func() {
		root, err := markdown.ParseString("- a\n  ```\n  x\n- b\n  ```")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(2))
		Expect(root.Children().FindDeep(content.IsOfType[*content.CodeBlock]())).To(BeNil())
	})
	FullyEqual("Code fence that only the next block closes", "- a\n  ```\n  x\n- b\n  ```")

	// Given a begin line that only an end line in the next block closes,
	// When written, Then the text is the same.
	FullyEqual("Quote begin line that only the next block closes", "- n\n  #+BEGIN_QUOTE\n  q\n- b\n  #+END_QUOTE")

	// Given a code fence that a fence line indented four more columns
	// closes, When parsed, Then the block has a code block with the line
	// between them.
	It("closes a code fence at a fence line that is indented four more columns", func() {
		root, err := markdown.ParseString("- a\n  ```\n  x\n      ```\n  after [[p]]")
		Expect(err).ToNot(HaveOccurred())

		code := root.Children().FindDeep(content.IsOfType[*content.CodeBlock]())
		Expect(code).ToNot(BeNil())
		Expect(code.(*content.CodeBlock).Code).To(Equal("x\n"))
		Expect(unclosedFencePageTitles(root.Blocks()[0])).To(ContainElement("p"))
	})
})
