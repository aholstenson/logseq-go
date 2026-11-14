package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A program can set the value of a text to a URL. Logseq makes a link from a
// bare URL, so the writer must escape a URL in a text, so that it reads back
// as text and not as a link.
var _ = Describe("Text that holds a URL", func() {
	write := func(node content.Node) string {
		v, err := markdown.AsString(node)
		Expect(err).ToNot(HaveOccurred())
		return v
	}

	readBack := func(output string) *content.Block {
		root, err := markdown.ParseString(output)
		Expect(err).ToNot(HaveOccurred())
		return root
	}

	// textOf joins the values of the texts, with a line break for each line
	// break of a text.
	textOf := func(root *content.Block) string {
		var b strings.Builder
		for _, n := range root.Children().FilterDeep(content.IsOfType[*content.Text]()) {
			text := n.(*content.Text)
			b.WriteString(text.Value)
			if text.SoftLineBreak || text.HardLineBreak {
				b.WriteString("\n")
			}
		}

		return b.String()
	}

	DescribeTable("reads back as text without a link",
		func(value string) {
			// Given a paragraph with the text
			paragraph := content.NewParagraph(content.NewText(value))

			// When it is written and read again
			output := write(paragraph)
			root := readBack(output)

			// Then it has no link and the same text
			Expect(root.Children().FindDeep(content.IsOfType[*content.AutoLink]())).To(BeNil(), "output: %q", output)
			Expect(textOf(root)).To(Equal(value), "output: %q", output)
		},
		Entry("a URL after a word", "see http://example.com"),
		Entry("a URL at the start", "https://example.com/a"),
		Entry("an FTP URL", "ftp://example.com"),
		Entry("a URL in parentheses", "(http://example.com)"),
		Entry("a URL that ends with a period", "go to http://example.com."),
		Entry("a URL on the second line", "a\nhttp://example.com"),
	)

	It("reads back a URL after emphasis as text", func() {
		// Given a paragraph of emphasis `a` and the text `http://example.com`
		paragraph := content.NewParagraph(content.NewEmphasis(content.NewText("a")), content.NewText("http://example.com"))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then it has no link
		Expect(root.Children().FindDeep(content.IsOfType[*content.AutoLink]())).To(BeNil(), "output: %q", output)
	})

	It("does not escape a URL after a letter", func() {
		// Given a paragraph with the text `xhttp://example.com`, which is not read as a link
		paragraph := content.NewParagraph(content.NewText("xhttp://example.com"))

		// When it is written, Then it is written as it is
		Expect(write(paragraph)).To(Equal("xhttp://example.com"))
	})

	It("does not escape a URL in the label of a link", func() {
		// Given a link with the label `http://example.com`
		paragraph := content.NewParagraph(content.NewLink("https://a.com", content.NewText("http://example.com")))

		// When it is written, Then the label is written as it is
		Expect(write(paragraph)).To(Equal("[http://example.com](https://a.com)"))
	})

	It("keeps a link that was read from a file", func() {
		// Given a block with a bare URL
		root := readBack("- see http://example.com")

		// When it is written, Then the URL is written as it is
		Expect(write(root)).To(Equal("- see http://example.com"))
	})
})
