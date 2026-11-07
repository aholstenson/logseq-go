package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A program can change the label of a link to a text with a `[` or `]`.
// A bracket that is not balanced in the label must be escaped, so that the
// link still reads back as a link with the same label.
var _ = Describe("Brackets in a changed link label", func() {
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

	setLabel := func(root *content.Block, value string) {
		text := root.Children().FindDeep(content.IsOfType[*content.Text]())
		Expect(text).ToNot(BeNil())
		text.(*content.Text).WithValue(value)
	}

	labelOf := func(node content.HasChildren) string {
		var b strings.Builder
		for _, n := range node.Children().FilterDeep(content.IsOfType[*content.Text]()) {
			b.WriteString(n.(*content.Text).Value)
		}

		return b.String()
	}

	It("escapes a closing bracket in the label of a link", func() {
		// Given a link with the label `a`
		root := parse("- [a](http://e.com)")

		// When the label is set to `a ] c` and the block is written
		setLabel(root, "a ] c")
		output := write(root)

		// Then the bracket is escaped
		Expect(output).To(Equal("- [a \\] c](http://e.com)"))

		// And it reads back as one link with the label `a ] c`
		links := parse(output).Children().FilterDeep(content.IsOfType[*content.Link]())
		Expect(links).To(HaveLen(1), "output: %q", output)
		Expect(labelOf(links[0].(*content.Link))).To(Equal("a ] c"))
	})

	It("keeps a labelled page link when an opening bracket is put in its label", func() {
		// Given a page link to `p` with the label `a`
		root := parse("- [a]([[p]])")

		// When the label is set to `a [ c` and the block is written and read again
		setLabel(root, "a [ c")
		output := write(root)
		links := parse(output).Children().FilterDeep(content.IsOfType[*content.PageLink]())

		// Then it reads back as one page link to `p` with the label `a [ c`
		Expect(links).To(HaveLen(1), "output: %q", output)
		link := links[0].(*content.PageLink)
		Expect(link.To).To(Equal("p"), "output: %q", output)
		Expect(labelOf(link)).To(Equal("a [ c"), "output: %q", output)
	})

	It("does not escape balanced brackets in the label of a link", func() {
		// Given a link with the label `a`
		root := parse("- [a](http://e.com)")

		// When the label is set to `a [b] c` and the block is written
		setLabel(root, "a [b] c")

		// Then the brackets are not escaped
		Expect(write(root)).To(Equal("- [a [b] c](http://e.com)"))
	})
})
