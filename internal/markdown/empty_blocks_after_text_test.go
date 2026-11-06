package markdown_test

import (
	"fmt"
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// emptyAfterTextOutline parses the input and gives one entry per block, as
// "<depth>:<content as written>". The pre-block has depth 0 and top-level
// blocks have depth 1.
func emptyAfterTextOutline(input string) []string {
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

// A bullet with no text is an empty block in Logseq, also when it comes
// right after a property line or after the text of the page.
var _ = Describe("Parsing empty blocks after properties or page text", func() {
	// Given a block with an id property and an empty child block, When
	// parsed, Then the empty child is a block and "-" is not text of a.
	It("reads an empty child block after an id property as a block", func() {
		Expect(emptyAfterTextOutline("- a\n  id:: 6650a7b2-0000-4000-8000-000000000001\n\t-\n- b\n")).To(Equal([]string{
			"1:a\nid:: 6650a7b2-0000-4000-8000-000000000001",
			"2:",
			"1:b",
		}))
	})

	It("keeps the id property of a block that has an empty child block", func() {
		root, err := markdown.ParseString("- a\n  id:: 6650a7b2-0000-4000-8000-000000000001\n\t-\n- b\n")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(2))
		a := root.Blocks()[0]
		Expect(a.ID()).To(Equal("6650a7b2-0000-4000-8000-000000000001"))
		Expect(a.Blocks()).To(HaveLen(1))
	})

	// Given a block with a property, an empty child and a grandchild, When
	// parsed, Then depths are 1, 2, 3.
	It("keeps the child of an empty block that comes after a property", func() {
		Expect(emptyAfterTextOutline("- a\n  key:: v\n\t- \n\t\t- c\n")).To(Equal([]string{
			"1:a\nkey:: v",
			"2:",
			"3:c",
		}))
	})

	// Given page properties followed by an empty block, When parsed, Then
	// the page has the pre-block, an empty block and block b.
	It("reads an empty block after page properties as a block", func() {
		Expect(emptyAfterTextOutline("title:: x\n-\n- b\n")).To(Equal([]string{
			"0:title:: x",
			"1:",
			"1:b",
		}))
	})

	// Given page text followed by an empty block with a child, When parsed
	// and written, Then the empty block comes before its child c.
	It("keeps the child of an empty block that comes after page text under it", func() {
		Expect(emptyAfterTextOutline("text\n-\n\t- c\n")).To(Equal([]string{
			"0:text",
			"1:",
			"2:c",
		}))
	})

	It("does not write the child of an empty block after page text before the empty block", func() {
		v := parseAndOutput("text\n-\n\t- c\n")
		Expect(v).ToNot(HavePrefix("text\n\t- c"))
	})
})
