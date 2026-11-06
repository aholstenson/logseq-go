package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A bullet that starts with "- " at the indentation of a child block is a
// child block in Logseq, also when the parent block ends with a list that
// uses "*" or "+" as its marker.
var _ = Describe("Parsing a child block after a star or plus list", func() {
	// Given a block whose content ends with a "*" list and a tab indented
	// child block, When parsed, Then the block has one child block and its
	// content holds a list with one item.
	It("reads a child block after a star list as a child block", func() {
		root, err := markdown.ParseString("- x\n  * a\n\t- child")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(1))
		x := root.Blocks()[0]
		Expect(x.Blocks()).To(HaveLen(1))
		Expect(markdown.AsString(x.Blocks()[0].Content()[0])).To(Equal("child"))

		list, ok := x.Children().FindDeep(content.IsOfType[*content.List]()).(*content.List)
		Expect(ok).To(BeTrue(), "the block holds a list")
		Expect(list.Children().Filter(content.IsOfType[*content.ListItem]())).To(HaveLen(1))
	})

	// Given the same page, When parsed and written, Then the text is the
	// same as the input.
	FullyEqual("Child block after a star list", "- x\n  * a\n\t- child")
	FullyEqual("Child block after a plus list", "- x\n  + a\n\t- child")

	// Given a block with a "*" list, a child block and a second top-level
	// block, When parsed, Then there are three blocks.
	It("reads a child block after a star list and the next top-level block as blocks", func() {
		root, err := markdown.ParseString("- x\n  * a\n\t- child\n- y")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(2))
		Expect(root.Blocks()[0].Blocks()).To(HaveLen(1))
	})

	DescribeTable("ends a list with another marker at a child block",
		func(input string, expected []string) {
			Expect(emptyParentOutline(input)).To(Equal(expected))
		},
		Entry("a child block indented with four spaces", "- x\n  * a\n    - child\n",
			[]string{"1:x\n* a", "2:child"}),
		Entry("a child block at the indentation of the list", "- x\n  * a\n  - child\n",
			[]string{"1:x\n* a", "2:child"}),
		Entry("a nested star list", "- x\n  * a\n    * b\n\t- child\n",
			[]string{"1:x\n* a\n  * b", "2:child"}),
		Entry("a numbered list", "- x\n  1. a\n\t- child\n",
			[]string{"1:x\n1. a", "2:child"}),
	)
})
