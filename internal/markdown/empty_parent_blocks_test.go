package markdown_test

import (
	"fmt"
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// emptyParentOutline parses the input and gives one entry per block, as
// "<depth>:<content as written>". Top-level blocks have depth 1.
func emptyParentOutline(input string) []string {
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

// Logseq writes a block without text as a bullet with no text. Such a block
// can have children like any other block.
var _ = Describe("Parsing empty blocks with children", func() {
	// Given a nested empty block with a child (tab indentation), When
	// parsed, Then the child stays under the empty block.
	It("keeps the child of a nested empty block under it", func() {
		Expect(emptyParentOutline("- a\n\t-\n\t\t- c\n")).To(Equal([]string{
			"1:a",
			"2:",
			"3:c",
		}))
	})

	// Given a nested empty block with a child at the end of a file with no
	// final newline, When parsed and written, Then the text of the child
	// is not joined to the text of the parent.
	It("does not join the child of a nested empty block to its parent at the end of a file", func() {
		input := "- a\n\t-\n\t\t- c"

		v := parseAndOutput(input)
		Expect(v).ToNot(ContainSubstring("a- c"))
		Expect(v).To(ContainSubstring("\t\t- c"))
	})

	// Given an empty top-level block with a child indented with two spaces,
	// When parsed, Then the child is under the empty block.
	It("keeps a child indented with spaces under an empty top-level block", func() {
		Expect(emptyParentOutline("-\n  - c\n")).To(Equal([]string{
			"1:",
			"2:c",
		}))
	})

	// Given a nested empty block (with a space after the bullet) with a
	// child, all indented with spaces, When parsed, Then depths are 1, 2, 3.
	It("keeps the child of a nested empty block indented with spaces under it", func() {
		Expect(emptyParentOutline("- a\n  - \n    - c\n")).To(Equal([]string{
			"1:a",
			"2:",
			"3:c",
		}))
	})

	// Given a nested empty block with two children and a sibling, When
	// parsed, Then only the two children are under the empty block.
	It("keeps several children of a nested empty block under it", func() {
		Expect(emptyParentOutline("- a\n\t-\n\t\t- b\n\t\t- c\n\t- d\n")).To(Equal([]string{
			"1:a",
			"2:",
			"3:b",
			"3:c",
			"2:d",
		}))
	})

	// Given a nested empty block with a child, When written, Then the child
	// is written one level deeper than the empty block.
	It("writes the child of a nested empty block one level deeper", func() {
		v := parseAndOutput("- a\n\t-\n\t\t- c\n")
		Expect(v).To(ContainSubstring("\t\t- c"))
	})

	// Given an empty block, a blank line and a child, When parsed, Then the
	// child is under the empty block.
	It("keeps a child after a blank line under an empty top-level block", func() {
		Expect(emptyParentOutline("-\n\n  - c\n")).To(Equal([]string{
			"1:",
			"2:c",
		}))
	})

	// Given a nested empty block with blank lines around it, When parsed,
	// Then the child is under the empty block, and the child is not code.
	It("keeps the child of a nested empty block after blank lines under it", func() {
		Expect(emptyParentOutline("- a\n\n\t-\n\n\t\t- c\n")).To(Equal([]string{
			"1:a",
			"2:",
			"3:c",
		}))
	})

	// Given an empty bullet indented two levels under a block, When parsed,
	// Then it is an empty child block, and not the text "-".
	It("reads an empty bullet indented two levels as a child block", func() {
		Expect(emptyParentOutline("- a\n\t\t-\n")).To(Equal([]string{
			"1:a",
			"2:",
		}))
	})

	// Given an empty block between two blocks, When parsed, Then there are
	// three blocks at the same level.
	It("reads an empty block between two blocks as a block", func() {
		Expect(emptyParentOutline("- a\n-\n- b\n")).To(Equal([]string{
			"1:a",
			"1:",
			"1:b",
		}))
	})

	// Given an empty top-level block with a tab indented child, When
	// parsed, Then the child is under the empty block (this works today).
	It("keeps a tab indented child under an empty top-level block", func() {
		Expect(emptyParentOutline("-\n\t- x\n")).To(Equal([]string{
			"1:",
			"2:x",
		}))
	})
})
