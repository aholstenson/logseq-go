package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Code that holds a ``` fence line must be saved with a fence that the code
// cannot close, so that the code and the text after it survive many saves.
var _ = Describe("Code blocks that hold a backtick fence", func() {
	codeOf := func(block *content.Block) []string {
		codes := []string{}
		for _, node := range block.Children().FilterDeep(content.IsOfType[*content.CodeBlock]()) {
			codes = append(codes, node.(*content.CodeBlock).Code)
		}
		return codes
	}

	pageRefTitles := func(block *content.Block) []string {
		titles := []string{}
		for _, ref := range block.Children().PageReferences() {
			titles = append(titles, ref.(content.PageRef).GetTo())
		}
		return titles
	}

	survivesTwoSaves := func(name string, input string) {
		It(name, func() {
			// Given a code block whose code holds a ``` fence, and text after it
			first, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())
			Expect(pageRefTitles(first)).To(ContainElement("q"))

			// When it is written, read, written and read again
			written, err := markdown.AsString(first)
			Expect(err).ToNot(HaveOccurred())
			second, err := markdown.ParseString(written)
			Expect(err).ToNot(HaveOccurred())
			writtenAgain, err := markdown.AsString(second)
			Expect(err).ToNot(HaveOccurred())
			third, err := markdown.ParseString(writtenAgain)
			Expect(err).ToNot(HaveOccurred())

			// Then the code is the same and the text after it is still a page ref
			Expect(codeOf(third)).To(Equal(codeOf(first)), "written twice: %q", writtenAgain)
			Expect(pageRefTitles(third)).To(ContainElement("q"), "written twice: %q", writtenAgain)
		})
	}

	survivesTwoSaves("keeps a four backtick block with a backtick fence inside",
		"- doc\n  ````md\n  ```js\n  x [[p]]\n  ```\n  ````\n  after [[q]]")
	survivesTwoSaves("keeps a tilde block with a backtick fence inside",
		"- t\n  ~~~\n  ```\n  inner\n  ```\n  ~~~\n  after [[q]]")
})
