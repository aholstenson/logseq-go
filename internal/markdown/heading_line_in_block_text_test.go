package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// headingLinePageText reads a page and writes it as a page is saved: the
// writer output with a line break at the end.
func headingLinePageText(input string) string {
	root, err := markdown.ParseString(input, markdown.AsPage())
	Expect(err).ToNot(HaveOccurred())
	v, err := markdown.AsString(root)
	Expect(err).ToNot(HaveOccurred())
	if v != "" && !strings.HasSuffix(v, "\n") {
		v += "\n"
	}
	return v
}

// headingInChildBlock gets the heading that is the first child of a child
// block of a top-level block, or nil.
func headingInChildBlock(root *content.Block) *content.Heading {
	for _, block := range root.Blocks() {
		for _, child := range block.Blocks() {
			if heading, ok := child.FirstChild().(*content.Heading); ok {
				return heading
			}
		}
	}
	return nil
}

// A heading line after the first line of a block, such as `## h` in
// "- t\n  ## h", is a child heading block without a bullet. Saving the
// block keeps the heading line at the column it was read at, so that every
// later save gives the same text and the heading stays a heading.
var _ = Describe("Parsing then outputting a heading line in the text of a block", func() {
	// Given a heading line and a text line under the text of a block,
	// When written, Then the text is the same.
	FullyEqual("Heading line and text line under the text of a block", "- t\n  ## h\n  t")

	// Given a heading line under the text of a block, When written, Then
	// the text is the same.
	FullyEqual("Level one heading line under the text of a block", "- a\n  # x")

	// Given a heading line with three spaces before it, When written, Then
	// the text is the same.
	FullyEqual("Heading line with three spaces before it", "- a\n   # h")

	// Given a heading line under an empty first line of a block, When
	// written, Then the heading line does not get a bullet.
	FullyEqual("Heading line under an empty first line of a block", "-\n  # x")

	// Given a tab-indented heading line, When written, read and written
	// again, Then both outputs are the same, the second read still has the
	// heading and the output has no escaped heading marks.
	It("gives the same text on a second save of a tab-indented heading line", func() {
		first := parseAndOutput("- t\n\t## h\n\tt")
		second := parseAndOutput(first)
		Expect(second).To(Equal(first))
		Expect(second).ToNot(ContainSubstring(`\##`))

		again, err := markdown.ParseString(second)
		Expect(err).ToNot(HaveOccurred())
		heading := again.Children().FindDeep(content.IsOfType[*content.Heading]())
		Expect(heading).ToNot(BeNil())
		Expect(heading.(*content.Heading).Level).To(Equal(2))
	})

	// Given a heading line and a text line, When written two times, Then
	// both outputs are the input.
	It("keeps a heading line and a text line over two saves", func() {
		input := "- a\n  ## x\n  y"
		first := parseAndOutput(input)
		Expect(first).To(Equal(input))
		Expect(parseAndOutput(first)).To(Equal(input))
	})

	// Given a heading line under the text of a block, When written two
	// times, Then both outputs are the input.
	It("keeps a heading line under the text of a block over two saves", func() {
		input := "- text\n  # heading line"
		first := parseAndOutput(input)
		Expect(first).To(Equal(input))
		second := parseAndOutput(first)
		Expect(second).To(Equal(input))

		again, err := markdown.ParseString(second)
		Expect(err).ToNot(HaveOccurred())
		heading := headingInChildBlock(again)
		Expect(heading).ToNot(BeNil())
		Expect(heading.Level).To(Equal(1))
	})

	// Given a heading line with a table under it, When written three
	// times, Then the third read still has the heading and the table, and
	// no output has escaped heading marks.
	It("keeps a table under a heading line over three saves", func() {
		v := "- x\n  ## h\n  | a | b |\n  | --- | --- |"
		for i := 0; i < 3; i++ {
			v = parseAndOutput(v)
			Expect(v).ToNot(ContainSubstring(`\##`), "save %d", i+1)
		}

		again, err := markdown.ParseString(v)
		Expect(err).ToNot(HaveOccurred())
		heading := headingInChildBlock(again)
		Expect(heading).ToNot(BeNil(), "text: %q", v)

		table := again.Children().FindDeep(content.IsOfType[*content.Table]())
		Expect(table).ToNot(BeNil(), "text: %q", v)
		cells := []string{}
		for _, cell := range table.(*content.Table).Header().Children() {
			value := ""
			for _, text := range cell.(*content.TableCell).Children().FilterDeep(content.IsOfType[*content.Text]()) {
				value += text.(*content.Text).Value
			}
			cells = append(cells, strings.TrimSpace(value))
		}
		Expect(cells).To(Equal([]string{"a", "b"}))
	})

	Describe("in a page", func() {
		// Given a page with a heading line under the text of a block, When
		// written, Then the line is not written with a tab.
		It("keeps a heading line under the text of a block", func() {
			input := "- text\n  # heading line\n"
			v := headingLinePageText(input)
			Expect(v).To(Equal(input))
			Expect(v).ToNot(ContainSubstring("\t# heading line"))
		})

		// Given a page with a level two heading line, When written two
		// times, Then both outputs are the input.
		It("keeps a level two heading line over two saves", func() {
			input := "- x\n  ## h\n"
			first := headingLinePageText(input)
			Expect(first).To(Equal(input))
			Expect(headingLinePageText(first)).To(Equal(input))
		})

		// Given a page with a heading line, When written three times, Then
		// each output is the input and the last read has the heading.
		It("keeps a heading line over three saves", func() {
			input := "- a\n  # x\n"
			v := input
			for i := 0; i < 3; i++ {
				v = headingLinePageText(v)
				Expect(v).To(Equal(input), "save %d", i+1)
			}

			again, err := markdown.ParseString(v, markdown.AsPage())
			Expect(err).ToNot(HaveOccurred())
			heading := headingInChildBlock(again)
			Expect(heading).ToNot(BeNil())
			Expect(heading.Level).To(Equal(1))
		})

		// Given a page with a blank line before a heading line, When
		// written, Then the blank line stays.
		It("keeps a blank line before a heading line", func() {
			input := "- x\n\n  ## h\n"
			Expect(headingLinePageText(input)).To(Equal(input))
		})

		// Given a page with a heading as a child bullet, When written,
		// Then the text is the same. This works now and must keep working.
		It("keeps a heading as a child bullet", func() {
			input := "- x\n\t- ## h\n"
			Expect(headingLinePageText(input)).To(Equal(input))
		})
	})
})
