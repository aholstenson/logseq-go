package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A `#` before a `#` does not start a tag, but a `#` before a backslash does.
// When the writer escapes the last `#` of a run, such as in `##b`, it must
// also escape the marks before it, so that the run reads back as text.
var _ = Describe("Text with a run of `#` marks before a word", func() {
	readBack := func(value string) (string, *content.Block) {
		output, err := markdown.AsString(content.NewParagraph(content.NewText(value)))
		Expect(err).ToNot(HaveOccurred())
		root, err := markdown.ParseString(output)
		Expect(err).ToNot(HaveOccurred())
		return output, root
	}

	textOf := func(root *content.Block) string {
		var b strings.Builder
		for _, n := range root.Children().FilterDeep(content.IsOfType[*content.Text]()) {
			b.WriteString(n.(*content.Text).Value)
		}

		return b.String()
	}

	DescribeTable("reads back as the same text without tags",
		func(value string) {
			// Given a paragraph with the text
			// When it is written and read again
			output, root := readBack(value)

			// Then it has no tag and the same text
			Expect(root.Children().FindDeep(content.IsOfType[*content.Hashtag]())).To(BeNil(), "output: %q", output)
			Expect(textOf(root)).To(Equal(value), "output: %q", output)
		},
		Entry("two marks", "x ##b"),
		Entry("two marks at the start", "##b"),
		Entry("three marks", "x ###b"),
		Entry("two marks before a page link", "x ##[[b]]"),
	)

	It("escapes a single mark once", func() {
		// Given a paragraph with the text `x #b`
		// When it is written, Then the mark is escaped once
		output, _ := readBack("x #b")
		Expect(output).To(Equal("x \\#b"))
	})
})
