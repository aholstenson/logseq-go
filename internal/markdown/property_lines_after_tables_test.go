package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// tableThenPropertyPageText reads a page and writes it as a page is saved:
// the writer output with a line break at the end.
func tableThenPropertyPageText(input string) string {
	root, err := markdown.ParseString(input, markdown.AsPage())
	Expect(err).ToNot(HaveOccurred())
	v, err := markdown.AsString(root)
	Expect(err).ToNot(HaveOccurred())
	if v != "" && !strings.HasSuffix(v, "\n") {
		v += "\n"
	}
	return v
}

// A property line after a table can be indented more than the table, as in
// "- y\n  | a |\n  | --- |\n   b:: 5". Logseq reads it as a property of
// the block. Reading gives the property, and saving does not add blank
// lines after it.
var _ = Describe("Parsing then outputting property lines after a table", func() {
	// Given a property line after a table with one more space, When it is
	// read and written, Then the block has the property and the text is
	// the same.
	It("reads a property line with one more space after a table", func() {
		input := "- y\n  | a |\n  | --- |\n   b:: 5\n- x"
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())

		blocks := root.Blocks()
		Expect(blocks).ToNot(BeEmpty())
		properties := blocks[0].FindProperties()
		Expect(properties).ToNot(BeNil())
		value := ""
		for _, node := range properties.Get("b").FilterDeep(content.IsOfType[*content.Text]()) {
			value += node.(*content.Text).Value
		}
		Expect(value).To(Equal("5"))

		Expect(parseAndOutput(input)).To(Equal(input))
	})

	// Given a page with a byte order mark, a table and a property line,
	// When it is written two times, Then both outputs are the input.
	It("does not add blank lines on two saves of a page with a byte order mark", func() {
		input := "\ufeff- [[P]]\n  | a | b |\n  | --- | --- |\n  rating:: 5\n\t- DONE x\n"
		first := tableThenPropertyPageText(input)
		Expect(first).To(Equal(input))
		Expect(tableThenPropertyPageText(first)).To(Equal(input))
	})

	// Given a page with a property line at the same indentation as the
	// table, When written, Then the text is the same. This works now and
	// must keep working.
	It("keeps a property line at the same indentation as the table", func() {
		input := "| a |\n| --- |\nb:: 5\n- x\n"
		Expect(tableThenPropertyPageText(input)).To(Equal(input))
	})
})
