package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A drawer can have a line that starts with `- `. Logseq reads the drawer up
// to its `:END:` line. Reading the page gives one drawer and no child block.
var _ = Describe("Parsing dash lines in drawers", func() {
	// Given a page with a drawer that has a `- ` line, When it is read,
	// Then the block has one drawer `mydrawer` and no child blocks.
	It("reads a dash line in a drawer as a part of the drawer", func() {
		root, err := markdown.ParseString("- x\n  :mydrawer:\n  stuff\n  - not bullet\n  :END:\n", markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())

		blocks := root.Blocks()
		Expect(blocks).To(HaveLen(1))
		Expect(blocks[0].Blocks()).To(BeEmpty())

		drawers := blocks[0].Children().FilterDeep(content.IsOfType[*content.Drawer]())
		Expect(drawers).To(HaveLen(1))
		Expect(drawers[0].(*content.Drawer).Name).To(Equal("mydrawer"))
	})
})
