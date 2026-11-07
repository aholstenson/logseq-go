package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Two lists of the same kind with only a blank line between them read as
// one list. When a program adds a new list right after a list, the written
// text must still read back as two lists.
var _ = Describe("A new list after a list of the same kind", func() {
	It("reads back as two lists", func() {
		// Given a block with a paragraph and an ordered list
		root, err := markdown.ParseString("- a\n  1. x")
		Expect(err).ToNot(HaveOccurred())

		// When a new ordered list is added to the block and the block is written
		root.Blocks()[0].AddChild(content.NewOrderedList(
			content.NewListItem(content.NewParagraph(content.NewText("p"))),
		))
		output, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())

		// Then the output reads back with two lists
		again, err := markdown.ParseString(output)
		Expect(err).ToNot(HaveOccurred())
		lists := again.Children().FilterDeep(content.IsOfType[*content.List]())
		Expect(lists).To(HaveLen(2), "output: %q", output)
	})
})
