package markdown_test

import (
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// When the last property of a block is removed, the block must be written
// as if it never had properties, with no blank line in their place.
var _ = Describe("Removing the last property of a block", func() {
	It("does not leave a blank line", func() {
		// Given two blocks, and the first block has one property
		root, err := markdown.ParseString("- a\n  k:: v\n- b")
		Expect(err).ToNot(HaveOccurred())

		// When the property is removed and the blocks are written
		root.Blocks()[0].Properties().Remove("k")
		output, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())

		// Then there is no blank line between the blocks
		Expect(output).To(Equal("- a\n- b"))
	})
})
