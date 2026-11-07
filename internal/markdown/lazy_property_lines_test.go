package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A property line after the property line of a bullet can be at column 0,
// as `baz:: q` is in "- foo:: bar\nbaz:: q". Logseq reads both lines as
// properties of the block. Reading the page gives one group of properties.
var _ = Describe("Parsing lazy property lines", func() {
	// Given a page with a property on the bullet line and a property line
	// at column 0, When it is read, Then the block has one group of
	// properties with both.
	It("reads a lazy property line in the same group", func() {
		root, err := markdown.ParseString("- foo:: bar\nbaz:: q\n", markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())

		blocks := root.Blocks()
		Expect(blocks).ToNot(BeEmpty())
		groups := blocks[0].Children().Filter(content.IsOfType[*content.Properties]())
		Expect(groups).To(HaveLen(1))
		properties := groups[0].(*content.Properties)
		Expect(properties.GetAsNode("foo")).ToNot(BeNil())
		Expect(properties.GetAsNode("baz")).ToNot(BeNil())
	})
})
