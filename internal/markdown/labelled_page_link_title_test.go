package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq reads `[label]([[Page]] "title")` as a ref to the page, with a
// title. The ref must be found and the title must be saved as written.
var _ = Describe("Labelled page links with a title", func() {
	It("reads the page ref of a labelled page link with a title", func() {
		// Given a labelled page link with a title in double quotes
		// When it is parsed
		block, err := markdown.ParseString(`- [lbl]([[a]] "t")`)
		Expect(err).ToNot(HaveOccurred())

		// Then the block has a page ref to `a`
		titles := []string{}
		for _, ref := range block.Children().PageReferences() {
			titles = append(titles, ref.(content.PageRef).GetTo())
		}
		Expect(titles).To(Equal([]string{"a"}))
	})

	It("saves a labelled page link with a title unchanged", func() {
		// Given a labelled page link with a title in double quotes
		input := `- [lbl]([[a]] "t")`

		// When it is parsed and written
		// Then the file text is unchanged
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	// pageRefTargets parses the input and returns the targets of all page
	// references in it.
	pageRefTargets := func(input string) []string {
		block, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())

		targets := []string{}
		for _, ref := range block.Children().PageReferences() {
			targets = append(targets, ref.(content.PageRef).GetTo())
		}
		return targets
	}

	// firstLink parses the input and returns the first page link or block
	// reference in it.
	firstLink := func(input string) content.Node {
		block, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())

		node := block.Children().FindDeep(content.IsEither(
			content.IsOfType[*content.PageLink](),
			content.IsOfType[*content.BlockRef](),
		))
		Expect(node).ToNot(BeNil())
		return node
	}

	It("reads the title of a labelled page link", func() {
		// Given a labelled page link with a title
		// When it is parsed
		link := firstLink(`- [lbl]([[a]] "t")`)

		// Then the link has the title
		Expect(link.(*content.PageLink).Title).To(Equal("t"))
	})

	It("reads a labelled page link with spaces in the name and a title", func() {
		// Given a page name with spaces and a title after two spaces
		input := `- [lbl]([[a b]]  "t x")`

		// Then the block has a page ref to the whole name
		Expect(pageRefTargets(input)).To(Equal([]string{"a b"}))

		// And the text is saved unchanged
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("reads a labelled block reference with a title", func() {
		// Given a labelled block reference with a title
		input := `- [lbl](((6578a3e0-1234-4abc-8def-0123456789ab)) "t")`

		// When it is parsed
		ref, ok := firstLink(input).(*content.BlockRef)

		// Then it is a block reference with the title
		Expect(ok).To(BeTrue())
		Expect(ref.ID).To(Equal("6578a3e0-1234-4abc-8def-0123456789ab"))
		Expect(ref.Title).To(Equal("t"))

		// And the text is saved unchanged
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	DescribeTable("keeps a link that Logseq reads as a search, not as a reference",
		func(input string) {
			// Then the block has no page ref
			Expect(pageRefTargets(input)).To(BeEmpty())

			// And the text is saved unchanged
			Expect(parseAndOutput(input)).To(Equal(input))
		},
		Entry("title in single quotes", `- [lbl]([[a]] 't')`),
		Entry("title in parentheses", `- [lbl]([[a]] (t))`),
		Entry("empty title", `- [lbl]([[a]] "")`),
	)

	It("writes a changed title in double quotes", func() {
		// Given a labelled page link with a title
		block, err := markdown.ParseString(`- [lbl]([[a]] "t")`)
		Expect(err).ToNot(HaveOccurred())

		// When the title is changed
		link := block.Children().FindDeep(content.IsOfType[*content.PageLink]()).(*content.PageLink)
		link.WithTitle(`say "hi"`)

		// Then the title is written in double quotes, the form that Logseq reads
		out, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal(`- [lbl]([[a]] "say \"hi\"")`))
	})

	It("keeps the title as written when the page is renamed", func() {
		// Given a labelled page link with a title after two spaces
		block, err := markdown.ParseString(`- [lbl]([[a]]  "t")`)
		Expect(err).ToNot(HaveOccurred())

		// When the link points at another page
		link := block.Children().FindDeep(content.IsOfType[*content.PageLink]()).(*content.PageLink)
		link.SetTo("b c")

		// Then only the page name changes
		out, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal(`- [lbl]([[b c]]  "t")`))
	})
})
