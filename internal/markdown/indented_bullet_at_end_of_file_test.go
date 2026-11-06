package markdown_test

import (
	"fmt"
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// endOfFileBlockText gives the content of the block as written, one node per
// line.
func endOfFileBlockText(b *content.Block) string {
	parts := []string{}
	for _, n := range b.Content() {
		s, err := markdown.AsString(n)
		Expect(err).ToNot(HaveOccurred())
		parts = append(parts, s)
	}
	return strings.Join(parts, "\n")
}

// endOfFileOutline gives one entry per block of the parsed input, as
// "<depth>:<content as written>". The content of the page before the first
// bullet has depth 0 and top-level blocks have depth 1.
func endOfFileOutline(root *content.Block) []string {
	out := []string{}
	if len(root.Content()) > 0 {
		out = append(out, "0:"+endOfFileBlockText(root))
	}
	var visit func(b *content.Block, depth int)
	visit = func(b *content.Block, depth int) {
		if b.IsPreBlock() {
			out = append(out, "0:"+endOfFileBlockText(b))
			for _, c := range b.Blocks() {
				visit(c, 1)
			}
			return
		}
		out = append(out, fmt.Sprintf("%d:%s", depth, endOfFileBlockText(b)))
		for _, c := range b.Blocks() {
			visit(c, depth+1)
		}
	}
	for _, b := range root.Blocks() {
		visit(b, 1)
	}
	return out
}

// endOfFileFindBlock finds the block, at any depth, whose content is the
// given text.
func endOfFileFindBlock(root *content.Block, text string) *content.Block {
	return root.Blocks().FindDeep(func(b *content.Block) bool {
		return endOfFileBlockText(b) == text
	})
}

// A file can end without a line break after its last line. An indented
// bullet on that last line is still a block.
var _ = Describe("Parsing an indented bullet on the last line of a file without a final line break", func() {
	// Given a heading with an indented bullet as the last line, When
	// parsed, Then the bullet is a block and there is no code block.
	It("reads an indented bullet after a heading as a block", func() {
		root, err := markdown.ParseString("# H\n\t- a")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Children().FindDeep(content.IsOfType[*content.CodeBlock]())).To(BeNil())
		Expect(endOfFileFindBlock(root, "a")).ToNot(BeNil())
	})

	// Given the same page, When written, Then no code fence is added.
	It("does not write an indented bullet after a heading as a code block", func() {
		v := parseAndOutput("# H\n\t- a")
		Expect(v).ToNot(ContainSubstring("```"))
	})

	// Given page text with an indented bullet as the last line, When
	// parsed, Then the page text has a child block a.
	It("reads an indented bullet after page text as a child block", func() {
		root, err := markdown.ParseString("text\n\t- a")
		Expect(err).ToNot(HaveOccurred())

		Expect(endOfFileOutline(root)).To(Equal([]string{
			"0:text",
			"1:a",
		}))
	})

	// Given a block with a bullet three levels deeper as the last line,
	// When parsed, Then that bullet is a child block of the first block.
	It("reads a bullet several levels deeper on the last line as a child block", func() {
		root, err := markdown.ParseString("- a\n\t\t\t- b")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(1))
		Expect(endOfFileBlockText(root.Blocks()[0])).To(Equal("a"))
		Expect(endOfFileFindBlock(root.Blocks()[0], "b")).ToNot(BeNil())
	})

	// Given page text with an indented bullet and a final line break, When
	// parsed, written and parsed again, Then both trees are the same.
	It("reads a written page with page text and an indented bullet back the same", func() {
		root, err := markdown.ParseString("text\n\t- a\n")
		Expect(err).ToNot(HaveOccurred())

		v, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())

		again, err := markdown.ParseString(v)
		Expect(err).ToNot(HaveOccurred())

		Expect(endOfFileOutline(again)).To(Equal(endOfFileOutline(root)), "written text: %q", v)
	})
})
