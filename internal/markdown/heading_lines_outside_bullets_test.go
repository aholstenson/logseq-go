package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// firstHeadingLevel gives the level of the heading that is the first node of
// the block, or 0 when the block does not start with a heading.
func firstHeadingLevel(b *content.Block) int {
	if len(b.Content()) == 0 {
		return 0
	}
	if h, ok := b.Content()[0].(*content.Heading); ok {
		return h.Level
	}
	return 0
}

// Logseq starts a new top-level block at each heading line that is not part
// of a bullet, such as "# Title" at the start of a line. Text and properties
// after the heading belong to that block. This is how the text of a page is read,
// so the tests parse with markdown.AsPage.
var _ = Describe("Parsing heading lines outside of bullets", func() {
	// Given a heading line between two bullets, When parsed, Then the
	// heading is a top-level block of its own.
	It("makes a heading line between two bullets a top-level block", func() {
		root, err := markdown.ParseString("- a\n# H\n- b", markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(3))
		Expect(firstHeadingLevel(root.Blocks()[1])).To(Equal(1))
	})

	// Given a page that starts with a heading line followed by a property,
	// When parsed, Then the heading is a block (not the pre-block) and the
	// property belongs to it.
	It("gives a property after a heading line at the start of a page to the heading block", func() {
		root, err := markdown.ParseString("# H\nkey:: v\n- a", markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(2))
		heading := root.Blocks()[0]
		Expect(heading.IsPreBlock()).To(BeFalse(), "a page that starts with a heading line has no pre-block")
		Expect(firstHeadingLevel(heading)).To(Equal(1))
		Expect(heading.Properties().Get("key")).ToNot(BeNil())
	})

	// Given two heading lines, When parsed, Then each is a block.
	It("makes each of two heading lines a block", func() {
		root, err := markdown.ParseString("# A\n# B", markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(2))
	})

	// Given a heading line with a bullet indented by two spaces after it,
	// When parsed, Then the bullet is a child of the heading block.
	It("makes a space indented bullet after a heading line a child of the heading block", func() {
		root, err := markdown.ParseString("# H\n  - beta", markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(1))
		Expect(firstHeadingLevel(root.Blocks()[0])).To(Equal(1))
		Expect(root.Blocks()[0].Blocks()).To(HaveLen(1))
	})

	// Given a heading line after a block with a child, When written, Then
	// the heading is not written into the first block before its child.
	It("does not write a heading line after a block with a child before that child", func() {
		v := parseAndOutput("- a\n\t- b\n# H\n- c")

		Expect(strings.Index(v, "# H")).To(BeNumerically(">", strings.Index(v, "\t- b")),
			"heading is written after block b, written text: %q", v)
	})
})
