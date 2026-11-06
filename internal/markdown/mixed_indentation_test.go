package markdown_test

import (
	"fmt"
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mixedIndentOutline parses the input and gives one entry per block, as
// "<depth>:<content as written>". Top-level blocks have depth 1.
func mixedIndentOutline(input string) []string {
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

// Logseq gives each bullet a level from the number of indentation
// characters before it, where a tab and a space each count as one. A bullet
// with a higher level than the bullet before it is a child of that bullet.
var _ = Describe("Parsing bullets indented with mixed or single spaces", func() {
	// Given a bullet indented with one space, When parsed, Then it is a
	// child of the bullet before it.
	It("makes a bullet indented with one space a child of the bullet before it", func() {
		Expect(mixedIndentOutline("- a\n - b\n- c")).To(Equal([]string{
			"1:a",
			"2:b",
			"1:c",
		}))
	})

	// Given a bullet indented with four spaces after a bullet indented with
	// a tab, When parsed, Then it is a child of the tab indented bullet.
	It("makes a bullet indented with four spaces a child of a bullet indented with a tab", func() {
		Expect(mixedIndentOutline("- x\n\t- y\n    - z")).To(Equal([]string{
			"1:x",
			"2:y",
			"3:z",
		}))
	})

	// Given a bullet indented with a tab after a bullet indented with two
	// spaces, When parsed, Then it is a sibling of the space indented
	// bullet.
	It("makes a bullet indented with a tab a sibling of a bullet indented with two spaces", func() {
		Expect(mixedIndentOutline("- a\n  - b\n\t- c")).To(Equal([]string{
			"1:a",
			"2:b",
			"2:c",
		}))
	})

	// Given a bullet indented with two spaces and a tab, When parsed, Then
	// it is a child block and not text of the block before it.
	It("reads a bullet indented with two spaces and a tab as a child block", func() {
		Expect(mixedIndentOutline("- a\n  \t- b\n")).To(Equal([]string{
			"1:a",
			"2:b",
		}))
	})

	// Given a bullet indented with two tabs after a bullet indented with
	// four spaces, When parsed, Then it is a sibling of the space indented
	// bullet.
	It("makes a bullet indented with two tabs a sibling of a bullet indented with four spaces", func() {
		Expect(mixedIndentOutline("- #tag\n    - TODO task\n\t\t- [[Page]]")).To(Equal([]string{
			"1:#tag",
			"2:TODO task",
			"2:[[Page]]",
		}))
	})
})
