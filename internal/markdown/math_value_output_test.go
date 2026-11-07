package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A program can set the value of inline math to a formula that has a `$`
// in it. The writer must write it in a form that reads back as the same
// math, or return an error.
var _ = Describe("Inline math with a dollar sign in its value", func() {
	It("writes math that reads back as one math node, or returns an error", func() {
		// Given a block with inline math
		root, err := markdown.ParseString("- $a$ x")
		Expect(err).ToNot(HaveOccurred())
		math := root.Children().FindDeep(content.IsOfType[*content.Math]())
		Expect(math).ToNot(BeNil())

		// When the value is set to `a $ b` and the block is written
		math.(*content.Math).WithValue("a $ b")
		output, err := markdown.AsString(root)
		if err != nil {
			// Then an error is a correct result
			return
		}

		// Then the output reads back as one math node with the same value
		again, err := markdown.ParseString(output)
		Expect(err).ToNot(HaveOccurred())
		maths := again.Children().FilterDeep(content.IsOfType[*content.Math]())
		Expect(maths).To(HaveLen(1), "output: %q", output)
		Expect(maths[0].(*content.Math).Value).To(Equal("a $ b"), "output: %q", output)
	})
})
