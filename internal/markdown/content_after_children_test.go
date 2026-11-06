package markdown_test

import (
	"fmt"
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// afterChildrenOutline parses the input and gives one entry per block, as
// "<depth>:<content nodes as written, one per line>". Top-level blocks have
// depth 1.
func afterChildrenOutline(input string) []string {
	root, err := markdown.ParseString(input)
	Expect(err).ToNot(HaveOccurred())

	out := []string{}
	var visit func(b *content.Block, depth int)
	visit = func(b *content.Block, depth int) {
		parts := []string{}
		for _, n := range b.Content() {
			s, err := markdown.AsString(n)
			Expect(err).ToNot(HaveOccurred())
			parts = append(parts, s)
		}
		out = append(out, fmt.Sprintf("%d:%s", depth, strings.Join(parts, "\n")))
		for _, c := range b.Blocks() {
			visit(c, depth+1)
		}
	}
	for _, b := range root.Blocks() {
		visit(b, 1)
	}
	return out
}

// Logseq adds text that is not indented and comes after a blank line to the
// block that comes last in the file, which can be a nested block.
var _ = Describe("Parsing text after the children of a block", func() {
	// Given a paragraph after a blank line that follows a child block, When
	// parsed, Then the paragraph is part of the child block.
	It("adds a paragraph after a child block to the child block", func() {
		Expect(afterChildrenOutline("- a\n\t- b\n\nc")).To(Equal([]string{
			"1:a",
			"2:b\nc",
		}))
	})

	// Given a paragraph after a blank line that follows a grandchild block,
	// When parsed, Then the paragraph is part of the grandchild block.
	It("adds a paragraph after a grandchild block to the grandchild block", func() {
		Expect(afterChildrenOutline("- a\n\t- b\n\t\t- d\n\nc\n")).To(Equal([]string{
			"1:a",
			"2:b",
			"3:d\nc",
		}))
	})

	// Given the same page, When written, Then the paragraph is not moved
	// before the child block.
	It("does not write a paragraph after a child block before the child block", func() {
		v := parseAndOutput("- a\n\t- b\n\nc")

		Expect(v).To(ContainSubstring("\t- b"))
		Expect(strings.Index(v, "c")).To(BeNumerically(">", strings.Index(v, "\t- b")),
			"text c is written after block b, written text: %q", v)
	})

	// Given a paragraph after the children of the second top-level block,
	// When parsed, Then the paragraph is part of the last child block.
	It("adds a paragraph to the last child of the last top-level block", func() {
		Expect(afterChildrenOutline("- a\n\t- b\n- x\n\t- y\n\nc\n")).To(Equal([]string{
			"1:a",
			"2:b",
			"1:x",
			"2:y\nc",
		}))
	})

	// Given a paragraph indented under a block, after a blank line that
	// follows a child of that block, When parsed, Then the paragraph is part
	// of the child block.
	It("adds an indented paragraph after a child block to the child block", func() {
		Expect(afterChildrenOutline("- a\n\t- b\n\n  c\n")).To(Equal([]string{
			"1:a",
			"2:b\nc",
		}))
	})

	// Given text before the first bullet with an indented child block, and a
	// paragraph after the child, When parsed and written, Then the paragraph
	// is part of the child block and is written after it.
	It("adds a paragraph after a child of the page text to that child", func() {
		block, err := markdown.ParseString("text\n\t- c\n\nmore\n")
		Expect(err).ToNot(HaveOccurred())

		preBlock := block.Blocks()[0]
		Expect(preBlock.Blocks()).To(HaveLen(1))

		child, err := markdown.AsString(preBlock.Blocks()[0])
		Expect(err).ToNot(HaveOccurred())
		Expect(child).To(ContainSubstring("more"))

		v := parseAndOutput("text\n\t- c\n\nmore\n")
		Expect(strings.Index(v, "more")).To(BeNumerically(">", strings.Index(v, "- c")), "written text: %q", v)
	})

	// Given a paragraph after a blank line that follows a block with no
	// children, When parsed, Then the paragraph stays in that block (this
	// works today).
	It("adds a paragraph after a block with no children to that block", func() {
		Expect(afterChildrenOutline("- a\n\nc")).To(Equal([]string{
			"1:a\nc",
		}))
	})
})
