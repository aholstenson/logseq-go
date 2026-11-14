package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A program can make a paragraph from more than one text. When the texts
// are written next to each other, the joined text must not become markup,
// such as a page link or a tag, that no single text has.
var _ = Describe("Texts next to each other that join into markup", func() {
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

	textOf := func(root *content.Block) string {
		var b strings.Builder
		for _, n := range root.Children().FilterDeep(content.IsOfType[*content.Text]()) {
			b.WriteString(n.(*content.Text).Value)
		}

		return b.String()
	}

	It("does not make a page link from two texts", func() {
		// Given a paragraph of the texts `[[a` and `]]`
		paragraph := content.NewParagraph(content.NewText("[[a"), content.NewText("]]"))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then the paragraph has no page link and its text is `[[a]]`
		Expect(root.Children().FindDeep(content.IsOfType[*content.PageLink]())).To(BeNil(), "output: %q", output)
		Expect(textOf(root)).To(Equal("[[a]]"), "output: %q", output)
	})

	It("does not make a tag from two texts", func() {
		// Given a paragraph of the texts `x #` and `a`
		paragraph := content.NewParagraph(content.NewText("x #"), content.NewText("a"))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then the paragraph has no tag
		Expect(root.Children().FindDeep(content.IsOfType[*content.Hashtag]())).To(BeNil(), "output: %q", output)
	})

	It("escapes emphasis markers that are split over two texts", func() {
		// Given a paragraph of the texts `*a` and `b*`
		paragraph := content.NewParagraph(content.NewText("*a"), content.NewText("b*"))

		// When it is written, Then both markers are escaped
		Expect(write(paragraph)).To(Equal("\\*ab\\*"))
	})

	It("does not make a macro from texts on both sides of emphasis", func() {
		// Given a paragraph of the text `{{a`, emphasis `x` and the text `}}`
		paragraph := content.NewParagraph(content.NewText("{{a"), content.NewEmphasis(content.NewText("x")), content.NewText("}}"))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then the paragraph has no macro, and keeps the emphasis
		Expect(root.Children().FindDeep(content.IsOfType[*content.Macro]())).To(BeNil(), "output: %q", output)
		Expect(root.Children().FindDeep(content.IsOfType[*content.Emphasis]())).ToNot(BeNil(), "output: %q", output)
		Expect(textOf(root)).To(Equal("{{ax}}"), "output: %q", output)
	})

	It("does not make code from backticks on both sides of a link", func() {
		// Given a paragraph of the text `` `a ``, a link and the text `` b` ``
		paragraph := content.NewParagraph(content.NewText("`a"), content.NewLink("u", content.NewText("l")), content.NewText("b`"))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then the paragraph has no code and keeps the link
		Expect(root.Children().FindDeep(content.IsOfType[*content.CodeSpan]())).To(BeNil(), "output: %q", output)
		Expect(root.Children().FindDeep(content.IsOfType[*content.Link]())).ToNot(BeNil(), "output: %q", output)
	})

	It("does not make code from a backtick in strong text and one after it", func() {
		// Given a paragraph of strong text `` `a `` and the text `` b` ``
		paragraph := content.NewParagraph(content.NewStrong(content.NewText("`a")), content.NewText("b`"))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then the paragraph has no code and keeps the strong text
		Expect(root.Children().FindDeep(content.IsOfType[*content.CodeSpan]())).To(BeNil(), "output: %q", output)
		Expect(root.Children().FindDeep(content.IsOfType[*content.Strong]())).ToNot(BeNil(), "output: %q", output)
	})

	It("does not make code from a backtick before a code span", func() {
		// Given a paragraph of the text `` `a ``, the code `x` and the text `` b` ``
		paragraph := content.NewParagraph(content.NewText("`a"), content.NewCodeSpan("x"), content.NewText("b`"))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then the paragraph has one code span, `x`
		spans := root.Children().FilterDeep(content.IsOfType[*content.CodeSpan]())
		Expect(spans).To(HaveLen(1), "output: %q", output)
		Expect(spans[0].(*content.CodeSpan).Value).To(Equal("x"), "output: %q", output)
	})

	It("does not make code from backticks on two lines", func() {
		// Given a paragraph of the line `` `a `` and the line `` b` ``
		first := content.NewText("`a")
		first.SoftLineBreak = true
		paragraph := content.NewParagraph(first, content.NewText("b`"))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then the paragraph has no code
		Expect(root.Children().FindDeep(content.IsOfType[*content.CodeSpan]())).To(BeNil(), "output: %q", output)
	})

	It("does not make math from dollar signs on both sides of emphasis", func() {
		// Given a paragraph of the text `$a`, emphasis `x` and the text `b$`
		paragraph := content.NewParagraph(content.NewText("$a"), content.NewEmphasis(content.NewText("x")), content.NewText("b$"))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then the paragraph has no math and keeps the emphasis
		Expect(root.Children().FindDeep(content.IsOfType[*content.Math]())).To(BeNil(), "output: %q", output)
		Expect(root.Children().FindDeep(content.IsOfType[*content.Emphasis]())).ToNot(BeNil(), "output: %q", output)
	})

	It("does not make math from a dollar sign before inline math", func() {
		// Given a paragraph of the text `$a`, the math `x` and the text `b$`
		paragraph := content.NewParagraph(content.NewText("$a"), content.NewMath("x"), content.NewText("b$"))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then the paragraph has one formula, `x`
		formulas := root.Children().FilterDeep(content.IsOfType[*content.Math]())
		Expect(formulas).To(HaveLen(1), "output: %q", output)
		Expect(formulas[0].(*content.Math).Value).To(Equal("x"), "output: %q", output)
	})

	It("does not make a tag from a `#` before emphasis", func() {
		// Given a paragraph of the text `#` and emphasis `x`
		paragraph := content.NewParagraph(content.NewText("#"), content.NewEmphasis(content.NewText("x")))

		// When it is written and read again
		output := write(paragraph)
		root := readBack(output)

		// Then the paragraph has no tag and keeps the emphasis
		Expect(root.Children().FindDeep(content.IsOfType[*content.Hashtag]())).To(BeNil(), "output: %q", output)
		Expect(root.Children().FindDeep(content.IsOfType[*content.Emphasis]())).ToNot(BeNil(), "output: %q", output)
	})

	It("does not escape `{{` that nothing after it closes", func() {
		// Given a paragraph of the text `{{a` and emphasis `x`
		paragraph := content.NewParagraph(content.NewText("{{a"), content.NewEmphasis(content.NewText("x")))

		// When it is written, Then the braces are not escaped
		Expect(write(paragraph)).To(Equal("{{a*x*"))
	})
})
