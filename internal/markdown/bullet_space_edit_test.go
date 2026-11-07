package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A block can have more than one space between its bullet and its content,
// for example `-   text`. When the content of such a block is changed, the
// written block must still read back with the same kind of content.
var _ = Describe("Edits in a block with spaces after its bullet", func() {
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

	It("keeps a code fence as a code block when its code is changed", func() {
		// Given a block with three spaces after the bullet and a code fence
		root := parse("-   ```\n    code\n    ```")
		code := root.Children().FindDeep(content.IsOfType[*content.CodeBlock]())
		Expect(code).ToNot(BeNil())

		// When the code is changed and the block is written
		code.(*content.CodeBlock).WithCode("code2\n")
		output := write(root)

		// Then the fence lines and the code line line up
		Expect(output).To(BeElementOf(
			"-   ```\n    code2\n    ```",
			"- ```\n  code2\n  ```",
		))

		// And the text reads back as a code block, not as a code span
		again := parse(output)
		readBack := again.Children().FindDeep(content.IsOfType[*content.CodeBlock]())
		Expect(readBack).ToNot(BeNil())
		Expect(readBack.(*content.CodeBlock).Code).To(Equal("code2\n"))
		Expect(again.Children().FindDeep(content.IsOfType[*content.CodeSpan]())).To(BeNil())
	})

	It("keeps a quote as one quote when a line of it is changed", func() {
		// Given a block with three spaces after the bullet and a quote of two lines
		root := parse("-   > q\n    > r")

		// When the text of the second line is changed and the block is written
		findText(root, "r").WithValue("R")
		output := write(root)

		// Then the text reads back as one quote with the lines q and R
		again := parse(output)
		quotes := again.Children().FilterDeep(content.IsOfType[*content.Blockquote]())
		Expect(quotes).To(HaveLen(1), "output: %q", output)
		Expect(quotes[0].(*content.Blockquote).Children().FindDeep(func(n content.Node) bool {
			t, ok := n.(*content.Text)
			return ok && t.Value == "q"
		})).ToNot(BeNil())
		Expect(quotes[0].(*content.Blockquote).Children().FindDeep(func(n content.Node) bool {
			t, ok := n.(*content.Text)
			return ok && t.Value == "R"
		})).ToNot(BeNil())
	})

	It("keeps a paragraph after spaces after the bullet when a line of it is changed", func() {
		// Given a block with three spaces after the bullet and a paragraph of two lines
		root := parse("-   text\n    more")

		// When the text of the second line is changed and the block is written
		findText(root, "more").WithValue("More")

		// Then only the changed word is different
		Expect(write(root)).To(Equal("-   text\n    More"))
	})
})
