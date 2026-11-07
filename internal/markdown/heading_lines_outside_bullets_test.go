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

// In the text of a block, a heading line before the first bullet is part of
// the block. After the first bullet, the text is an outline of blocks, so a
// heading line starts a top-level block, the same as in a page.
var _ = Describe("Parsing heading lines after the bullets in the text of a block", func() {
	It("makes a heading task line after a bullet a block of its own", func() {
		// Given block text with a bullet and then a heading task line
		root, err := markdown.ParseString("- a\n## TODO b")
		Expect(err).ToNot(HaveOccurred())

		// Then the heading is a second top-level block, and a TODO task
		Expect(root.Blocks()).To(HaveLen(2))
		Expect(root.Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusNone))
		Expect(root.Blocks()[0].Blocks()).To(BeEmpty())
		Expect(root.Blocks()[1].TaskStatus()).To(Equal(content.TaskStatusTodo))
		Expect(firstHeadingLevel(root.Blocks()[1])).To(Equal(2))
	})

	It("makes a heading line after a nested bullet a top-level block", func() {
		// Given block text where the last bullet before the heading is nested
		root, err := markdown.ParseString("- a\n\t- a1\n## b")
		Expect(err).ToNot(HaveOccurred())

		// Then the heading is a top-level block, and not a child of a or a1
		Expect(root.Blocks()).To(HaveLen(2))
		Expect(root.Blocks()[0].Blocks()).To(HaveLen(1))
		Expect(root.Blocks()[0].Blocks()[0].Blocks()).To(BeEmpty())
		Expect(firstHeadingLevel(root.Blocks()[1])).To(Equal(2))
	})

	It("keeps a heading line before the first bullet in the block", func() {
		// Given block text that starts with a heading line and then a bullet
		root, err := markdown.ParseString("# T\ntext\n- a")
		Expect(err).ToNot(HaveOccurred())

		// Then the heading is content of the block, and the bullet is its
		// only child block
		Expect(firstHeadingLevel(root)).To(Equal(1))
		Expect(root.Blocks()).To(HaveLen(1))
	})

	// Given block text with a heading line after the bullets, When written,
	// Then the text is the same as the input.
	FullyEqual("Heading task line after a bullet", "- a\n## TODO b")
	FullyEqual("Heading task line between two bullets", "- a\n## TODO b\n- c")
	FullyEqual("Heading line after a nested bullet", "- a\n\t- a1\n## TODO b")
	FullyEqual("Heading line after bullets indented under text", "root\n\t- child\n## h")
})
