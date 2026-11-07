package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The spaces at the end of a line are kept as they are in the file. When a
// program changes the kind of line break of a text, or changes its value,
// the written line break must be the kind that the text has now.
var _ = Describe("Line breaks of a text after it is changed", func() {
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

	firstText := func(root *content.Block) *content.Text {
		text := root.Children().FindDeep(content.IsOfType[*content.Text]())
		Expect(text).ToNot(BeNil())
		return text.(*content.Text)
	}

	It("writes a hard line break when a soft line break with one trailing space is changed to a hard one", func() {
		// Given a line with one space at its end and a soft line break
		root := parse("- a \n  b")

		// When the line break is changed to a hard line break and the block is written and read again
		firstText(root).WithHardLineBreak()
		output := write(root)

		// Then the text reads back with a hard line break
		Expect(firstText(parse(output)).HardLineBreak).To(BeTrue(), "output: %q", output)
	})

	It("keeps a soft line break when the value gets a space at its end", func() {
		// Given a line with one space at its end and a soft line break
		root := parse("- a \n  b")

		// When the value is set to `a ` and the block is written and read again
		firstText(root).WithValue("a ")
		output := write(root)

		// Then the text reads back with a soft line break
		Expect(firstText(parse(output)).SoftLineBreak).To(BeTrue(), "output: %q", output)
	})
})
