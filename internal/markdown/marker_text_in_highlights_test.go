package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A text at the start or the end of a strikethrough or a highlight is written
// next to the markers of that node. When the text starts or ends with the
// marker character, that character would join the run of markers, as in the
// highlight of `^` that would be written as `^^^^^`. The writer must escape it.
var _ = Describe("Marker characters at the edge of a strikethrough or a highlight", func() {
	equalSigns := func(nodes ...content.Node) content.Node {
		highlight := content.NewHighlight(nodes...)
		highlight.EqualSigns = true
		return highlight
	}

	textOf := func(root *content.Block) string {
		var b strings.Builder
		for _, n := range root.Children().FilterDeep(content.IsOfType[*content.Text]()) {
			b.WriteString(n.(*content.Text).Value)
		}

		return b.String()
	}

	DescribeTable("reads back as the same node and text",
		func(node content.Node, value string) {
			// Given a paragraph with the node between two words
			paragraph := content.NewParagraph(content.NewText("a "), node, content.NewText(" b"))

			// When it is written and read again
			output, err := markdown.AsString(paragraph)
			Expect(err).ToNot(HaveOccurred())
			root, err := markdown.ParseString(output)
			Expect(err).ToNot(HaveOccurred())

			// Then it has a node of the same type with the same text
			found := root.Children().FindDeep(func(n content.Node) bool {
				switch n.(type) {
				case *content.Highlight, *content.Strikethrough:
					return true
				}
				return false
			})
			Expect(found).ToNot(BeNil(), "output: %q", output)
			Expect(found).To(BeAssignableToTypeOf(node), "output: %q", output)
			Expect(textOf(root)).To(Equal("a "+value+" b"), "output: %q", output)
		},
		Entry("a highlight of `^`", content.NewHighlight(content.NewText("^")), "^"),
		Entry("a highlight of `=` with equal signs", equalSigns(content.NewText("=")), "="),
		Entry("a strikethrough of `~`", content.NewStrikethrough(content.NewText("~")), "~"),
		Entry("a highlight that ends with ` ^`", content.NewHighlight(content.NewText("x ^")), "x ^"),
		Entry("a strikethrough that starts with `~ `", content.NewStrikethrough(content.NewText("~ x")), "~ x"),
	)

	It("keeps a highlight from a file as it is", func() {
		// Given a block with a highlight that ends with `^`
		root, err := markdown.ParseString("- ^^a^^^")
		Expect(err).ToNot(HaveOccurred())

		// When it is written, Then it is written as it is
		Expect(markdown.AsString(root)).To(Equal("- ^^a^^^"))
	})
})
