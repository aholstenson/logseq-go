package markdown_test

import (
	"fmt"
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// skippedLevelsOutline parses the input and gives one entry per block, as
// "<depth>:<content as written>". A pre-block has depth 0 and top-level
// blocks have depth 1.
func skippedLevelsOutline(input string) []string {
	root, err := markdown.ParseString(input)
	Expect(err).ToNot(HaveOccurred())

	text := func(b *content.Block) string {
		parts := []string{}
		for _, n := range b.Content() {
			s, err := markdown.AsString(n)
			Expect(err).ToNot(HaveOccurred())
			parts = append(parts, s)
		}
		return strings.Join(parts, "\n")
	}

	out := []string{}
	if len(root.Content()) > 0 {
		out = append(out, "0:"+text(root))
	}
	var visit func(b *content.Block, depth int)
	visit = func(b *content.Block, depth int) {
		if b.IsPreBlock() {
			out = append(out, "0:"+text(b))
			for _, c := range b.Blocks() {
				visit(c, 1)
			}
			return
		}
		out = append(out, fmt.Sprintf("%d:%s", depth, text(b)))
		for _, c := range b.Blocks() {
			visit(c, depth+1)
		}
	}
	for _, b := range root.Blocks() {
		visit(b, 1)
	}
	return out
}

// Logseq nests bullets by comparing the indentation of a bullet with the
// bullets before it. When indentation levels are skipped, or a page starts
// with an indented bullet, this gives a flatter tree than the visible
// indentation.
var _ = Describe("Parsing bullets that skip indentation levels", func() {
	// Given a block with children that skip levels and then go back one
	// level, When parsed, Then all of them are children of the first block.
	It("makes bullets that skip levels and go back one level children of the first block", func() {
		Expect(skippedLevelsOutline("- a\n\t\t\t- b\n\t\t- c\n\t\t\t- d\n")).To(Equal([]string{
			"1:a",
			"2:b",
			"2:c",
			"2:d",
		}))
	})

	// Given a page that starts with a tab indented bullet, When parsed, Then
	// there are three top-level blocks and no pre-block.
	It("makes all bullets top-level blocks when a page starts with a tab indented bullet", func() {
		Expect(skippedLevelsOutline("\t- a\n- b\n\t- c\n")).To(Equal([]string{
			"1:a",
			"1:b",
			"1:c",
		}))
	})

	// Given a page that starts with a bullet indented with two spaces, When
	// parsed, Then there are three top-level blocks.
	It("makes all bullets top-level blocks when a page starts with a space indented bullet", func() {
		Expect(skippedLevelsOutline("  - x\n- y\n  - z\n")).To(Equal([]string{
			"1:x",
			"1:y",
			"1:z",
		}))
	})
})
