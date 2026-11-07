package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A program can set the value of a text to anything, also to text that
// looks like block syntax, such as `- b`, `1. b`, `> b` or `k:: v`. When
// such a text starts a line, the writer must escape it, so that it reads
// back as the same text and not as a list, a block, a quote or a property.
var _ = Describe("Text that starts a line and looks like block syntax", func() {
	parse := func(input string) *content.Block {
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		return root
	}

	write := func(root *content.Block) string {
		v, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())
		return v
	}

	findText := func(root *content.Block, value string) *content.Text {
		for _, node := range root.Children().FilterDeep(content.IsOfType[*content.Text]()) {
			if text := node.(*content.Text); text.Value == value {
				return text
			}
		}

		Fail("no text " + value)
		return nil
	}

	// textOf joins the values of the texts in the node, with a line break
	// for each line break of a text.
	textOf := func(node content.HasChildren) string {
		var b strings.Builder
		for _, n := range node.Children().FilterDeep(content.IsOfType[*content.Text]()) {
			text := n.(*content.Text)
			b.WriteString(text.Value)
			if text.SoftLineBreak || text.HardLineBreak {
				b.WriteString("\n")
			}
		}

		return strings.TrimSuffix(b.String(), "\n")
	}

	// readBackOneBlock writes the root, reads the output again and checks
	// that it has one block without child blocks.
	readBackOneBlock := func(root *content.Block) (string, *content.Block) {
		output := write(root)
		again := parse(output)
		Expect(again.Blocks()).To(HaveLen(1), "output: %q", output)
		block := again.Blocks()[0]
		Expect(block.Blocks()).To(BeEmpty(), "output: %q", output)
		return output, block
	}

	It("keeps a dash after a soft line break as text", func() {
		// Given a block with a paragraph of two lines
		root := parse("- a\n  b")

		// When the second text is set to `- b` and the block is written and read again
		findText(root, "b").WithValue("- b")
		output, block := readBackOneBlock(root)

		// Then the block has no child block, and its text is `a`, a line break and `- b`
		paragraph := block.Children().FindDeep(content.IsOfType[*content.Paragraph]())
		Expect(paragraph).ToNot(BeNil(), "output: %q", output)
		Expect(textOf(paragraph.(*content.Paragraph))).To(Equal("a\n- b"), "output: %q", output)
	})

	It("keeps text that looks like a property after a soft line break as text", func() {
		// Given a block with a paragraph of two lines
		root := parse("- a\n  b")

		// When the second text is set to `b:: c` and the block is written and read again
		findText(root, "b").WithValue("b:: c")
		output, block := readBackOneBlock(root)

		// Then the block has no property `b`
		if properties := block.FindProperties(); properties != nil {
			Expect(properties.Get("b")).To(BeEmpty(), "output: %q", output)
		}
	})

	It("keeps first-line text that looks like an ordered list as text", func() {
		// Given a block with one line of text
		root := parse("- a")

		// When the text is set to `1. b` and the block is written and read again
		findText(root, "a").WithValue("1. b")
		output, block := readBackOneBlock(root)

		// Then the block has the text `1. b` and no list
		Expect(block.Children().FindDeep(content.IsOfType[*content.List]())).To(BeNil(), "output: %q", output)
		Expect(textOf(block)).To(Equal("1. b"), "output: %q", output)
	})

	It("keeps a value `1. b` on a continuation line as text", func() {
		// Given a block with a paragraph of two lines
		root := parse("- a\n  b")

		// When the second text is set to `1. b` and the block is written and read again
		findText(root, "b").WithValue("1. b")
		output, block := readBackOneBlock(root)

		// Then it reads back as one paragraph with the text `1. b` on the second line
		Expect(block.Children().FindDeep(content.IsOfType[*content.List]())).To(BeNil(), "output: %q", output)
		paragraphs := block.Children().FilterDeep(content.IsOfType[*content.Paragraph]())
		Expect(paragraphs).To(HaveLen(1), "output: %q", output)
		Expect(textOf(paragraphs[0].(*content.Paragraph))).To(Equal("a\n1. b"), "output: %q", output)
	})

	It("keeps a value `k:: v` on the first line as text", func() {
		// Given a block with one line of text
		root := parse("- a")

		// When the text is set to `k:: v` and the block is written and read again
		findText(root, "a").WithValue("k:: v")
		output, block := readBackOneBlock(root)

		// Then the block has no properties and its paragraph text is `k:: v`
		Expect(block.FindProperties()).To(BeNil(), "output: %q", output)
		paragraph := block.Children().FindDeep(content.IsOfType[*content.Paragraph]())
		Expect(paragraph).ToNot(BeNil(), "output: %q", output)
		Expect(textOf(paragraph.(*content.Paragraph))).To(Equal("k:: v"), "output: %q", output)
	})

	It("keeps the text of a task as text when the task marker is removed", func() {
		// Given a task whose text starts with `- `
		root := parse("- TODO - x")

		// When the task marker is removed and the block is written and read again
		root.Blocks()[0].SetTaskStatus(content.TaskStatusNone)
		output, block := readBackOneBlock(root)

		// Then it is one block with the text `- x`
		Expect(textOf(block)).To(Equal("- x"), "output: %q", output)
	})

	It("escapes a heading marker on the first line", func() {
		// Given a block with one line of text
		root := parse("- a")

		// When the text is set to `# b` and the block is written
		findText(root, "a").WithValue("# b")

		// Then the heading marker is escaped
		Expect(write(root)).To(Equal("- \\# b"))
	})

	It("escapes a heading marker on a continuation line once", func() {
		// Given a block with a paragraph of two lines
		root := parse("- a\n  b")

		// When the second text is set to `# b` and the block is written
		findText(root, "b").WithValue("# b")

		// Then the heading marker is escaped once
		Expect(write(root)).To(Equal("- a\n  \\# b"))
	})
})
