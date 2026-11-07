package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// tabColumnPageText reads a page and writes it as a page is saved: the
// writer output with a line break at the end.
func tabColumnPageText(input string) string {
	root, err := markdown.ParseString(input, markdown.AsPage())
	Expect(err).ToNot(HaveOccurred())
	v, err := markdown.AsString(root)
	Expect(err).ToNot(HaveOccurred())
	if v != "" && !strings.HasSuffix(v, "\n") {
		v += "\n"
	}
	return v
}

// tabColumnTable gets the first table of a node.
func tabColumnTable(root *content.Block) *content.Table {
	table := root.Children().FindDeep(content.IsOfType[*content.Table]())
	Expect(table).ToNot(BeNil())
	return table.(*content.Table)
}

// tabColumnCells gets the text of each cell of a table row.
func tabColumnCells(row *content.TableRow) []string {
	cells := []string{}
	for _, cell := range row.Children() {
		value := ""
		for _, text := range cell.(*content.TableCell).Children().FilterDeep(content.IsOfType[*content.Text]()) {
			value += text.(*content.Text).Value
		}
		cells = append(cells, strings.TrimSpace(value))
	}
	return cells
}

// tabColumnBodyCells gets the cells of the first body row of the first
// table of a node.
func tabColumnBodyCells(root *content.Block) []string {
	rows := tabColumnTable(root).BodyRows()
	Expect(rows).ToNot(BeEmpty())
	return tabColumnCells(rows[0])
}

// A tab before a line can start before the column where the text of a list
// item, quote or block starts, as the tab in "- 1. | a |\n\t|---|" does. The
// part of the tab after that column is spaces of the line. Reading such a
// table or footnote keeps the text of its cells, and saving keeps it.
var _ = Describe("Parsing then outputting lines with a tab across the column of a container", func() {
	Describe("tables", func() {
		// Given a page with a table under an ordered list item whose lines
		// start with a tab, When it is read and written, Then the body
		// cells are 1 and 2 and no cell is written as an escaped pipe.
		It("keeps the cells of a table under a tab-indented ordered list item", func() {
			input := "- 1. | a | b |\n\t|---|---|\n\t| 1 | 2 |\n- n\n"
			root, err := markdown.ParseString(input, markdown.AsPage())
			Expect(err).ToNot(HaveOccurred())
			Expect(tabColumnBodyCells(root)).To(Equal([]string{"1", "2"}))

			v := tabColumnPageText(input)
			Expect(v).ToNot(ContainSubstring(`\|`))
			again, err := markdown.ParseString(v, markdown.AsPage())
			Expect(err).ToNot(HaveOccurred())
			Expect(tabColumnBodyCells(again)).To(Equal([]string{"1", "2"}))
		})

		// Given a table at the end of the text whose lines start with a
		// tab, When it is read, Then there is no panic and the body cells
		// are 1, 2 and 3.
		It("reads a table at the end of the text without a panic", func() {
			var root *content.Block
			Expect(func() {
				var err error
				root, err = markdown.ParseString("-  a | b\n\t-|-\n\t1 | 2 | 3")
				Expect(err).ToNot(HaveOccurred())
			}).ToNot(Panic())
			Expect(tabColumnBodyCells(root)).To(Equal([]string{"1", "2", "3"}))
		})

		// Given a table at the end of the text under an ordered list item,
		// When it is read and written, Then the output has no NUL byte and
		// the cells read back as 1 and 2.
		It("writes no NUL byte for a table at the end of the text", func() {
			v := parseAndOutput("- 1. | a | b |\n\t|---|---|\n\t| 1 | 2 |")
			Expect(v).ToNot(ContainSubstring("\x00"))

			again, err := markdown.ParseString(v)
			Expect(err).ToNot(HaveOccurred())
			Expect(tabColumnBodyCells(again)).To(Equal([]string{"1", "2"}))
		})

		// Given a page with a table in a quote with a tab after each `>`,
		// When it is read, Then the body cells are 1 and 2.
		It("keeps the cells of a table in a quote with a tab after the marker", func() {
			root, err := markdown.ParseString(">\t| a | b |\n>\t|---|---|\n>\t| 1 | 2 |\n", markdown.AsPage())
			Expect(err).ToNot(HaveOccurred())
			Expect(tabColumnBodyCells(root)).To(Equal([]string{"1", "2"}))

			again, err := markdown.ParseString(tabColumnPageText(">\t| a | b |\n>\t|---|---|\n>\t| 1 | 2 |\n"), markdown.AsPage())
			Expect(err).ToNot(HaveOccurred())
			Expect(tabColumnBodyCells(again)).To(Equal([]string{"1", "2"}))
		})

		// Given a table under the text of a block whose lines start with a
		// tab, When it is read and written, Then the cells are a, b, 1 and
		// 2 and no cell is written as an escaped pipe.
		It("keeps the cells of a table under a tab-indented line of a block", func() {
			input := "- x\n\t| a | b |\n\t| --- | --- |\n\t| 1 | 2 |"
			root, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())
			Expect(tabColumnCells(tabColumnTable(root).Header())).To(Equal([]string{"a", "b"}))
			Expect(tabColumnBodyCells(root)).To(Equal([]string{"1", "2"}))
			Expect(parseAndOutput(input)).ToNot(ContainSubstring(`\|`))
		})

		// Given a page with a table in a child block whose lines start
		// with two tabs, When it is written and read again, Then the
		// header cells are a and b.
		It("keeps the header cells of a table under two tabs in a child block", func() {
			v := tabColumnPageText("- a\n\t- y\n\t\t| a | b |\n\t\t| --- | --- |\n")
			again, err := markdown.ParseString(v, markdown.AsPage())
			Expect(err).ToNot(HaveOccurred())
			Expect(tabColumnCells(tabColumnTable(again).Header())).To(Equal([]string{"a", "b"}), "text: %q", v)
		})

		// Given a page with a table under an ordered list item whose lines
		// start with spaces, When written, Then the text is the same. This
		// works now and must keep working.
		It("keeps a table under an ordered list item indented with spaces", func() {
			input := "- 1. | a | b |\n     |---|---|\n     | 1 | 2 |\n"
			Expect(tabColumnPageText(input)).To(Equal(input))
		})

		// Given a table in a child block whose lines start with a tab and
		// two spaces, When written, Then the text is the same. This works
		// now and must keep working.
		FullyEqual("Table under a tab and two spaces in a child block", "- a\n\t- y\n\t  | a | b |\n\t  | --- | --- |")
	})

	Describe("footnotes", func() {
		// Given a footnote definition on a tab-indented line of a block,
		// When it is read and written, Then its text is `a` and the label
		// is not repeated in the text.
		It("keeps the text of a footnote definition after a tab", func() {
			input := "- x\n\t[^1]: a"
			root, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			definition := root.Children().FindDeep(content.IsOfType[*content.FootnoteDefinition]())
			Expect(definition).ToNot(BeNil())
			text := ""
			for _, node := range definition.(*content.FootnoteDefinition).Children().FilterDeep(content.IsOfType[*content.Text]()) {
				text += node.(*content.Text).Value
			}
			Expect(text).To(Equal("a"))
			Expect(parseAndOutput(input)).ToNot(ContainSubstring("]: ]: a"))
		})
	})
})
