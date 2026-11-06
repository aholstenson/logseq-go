package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A saved block must keep the emphasis and rule characters that the user
// wrote, even when another character has the same meaning.
var _ = Describe("Emphasis and rule markers in saved blocks", func() {
	keepsMarkers := func(name string, input string) {
		It(name, func() {
			// Given a block with underscore or asterisk markers
			// When it is parsed and written
			// Then the file text is unchanged
			Expect(parseAndOutput(input)).To(Equal(input))
		})
	}

	keepsMarkers("keeps double underscore strong emphasis", "- __init__")
	keepsMarkers("keeps single underscore emphasis", "- __init__ and _a_")
	keepsMarkers("keeps a rule made of underscores", "- ___")
	keepsMarkers("keeps a rule made of asterisks", "- ***")

	keepsMarkers("keeps underscores around strong and emphasis together", "- ___x___")
	keepsMarkers("keeps underscores of emphasis inside strong text", "- __a _b_ c__")
	keepsMarkers("keeps underscores around a page link", "- __[[x]]__")
	keepsMarkers("keeps underscores inside parentheses", "- (_a_)")
	keepsMarkers("keeps both markers in the same block", "- *a* and _b_ and **c** and __d__")
	keepsMarkers("keeps a rule with spaces between the characters", "- * * *")
	keepsMarkers("keeps a rule with more than three characters", "- ****")
	keepsMarkers("keeps a rule outside a bullet", "text\n\n___\n\nmore")
	keepsMarkers("keeps a rule in the text of a block", "- a\n  ***\n  b")

	It("reads which marker emphasis and strong text are written with", func() {
		// Given emphasis and strong text with each marker
		block, err := markdown.ParseString("_a_ *b* __c__ **d**")
		Expect(err).ToNot(HaveOccurred())

		// When it is parsed
		emphasis := block.Children().FilterDeep(content.IsOfType[*content.Emphasis]())
		strong := block.Children().FilterDeep(content.IsOfType[*content.Strong]())

		// Then only the nodes written with underscores are marked so
		Expect(emphasis).To(HaveLen(2))
		Expect(emphasis[0].(*content.Emphasis).Underscore).To(BeTrue())
		Expect(emphasis[1].(*content.Emphasis).Underscore).To(BeFalse())
		Expect(strong).To(HaveLen(2))
		Expect(strong[0].(*content.Strong).Underscore).To(BeTrue())
		Expect(strong[1].(*content.Strong).Underscore).To(BeFalse())
	})

	It("writes asterisks for underscore emphasis directly after a letter", func() {
		// Given underscore emphasis directly after a letter, where `_` would
		// not be read back as emphasis
		paragraph := content.NewParagraph(
			content.NewText("a"),
			content.NewEmphasis(content.NewText("b")).WithUnderscore(true),
			content.NewText(" c"),
		)

		// When it is written
		v, err := markdown.AsString(paragraph)
		Expect(err).ToNot(HaveOccurred())

		// Then asterisks are written, so it stays emphasis
		Expect(v).To(Equal("a*b* c"))
	})

	It("writes asterisks for underscore strong text directly before a letter", func() {
		// Given underscore strong text directly before a letter
		paragraph := content.NewParagraph(
			content.NewStrong(content.NewText("a")).WithUnderscore(true),
			content.NewText("b"),
		)

		// When it is written
		v, err := markdown.AsString(paragraph)
		Expect(err).ToNot(HaveOccurred())

		// Then asterisks are written, so it stays strong text
		Expect(v).To(Equal("**a**b"))
	})

	It("writes a rule with a marker that is not a rule as three dashes", func() {
		// Given a thematic break with a marker that is not a thematic break
		rule := content.NewThematicBreak().WithMarker("abc")

		// When it is written
		v, err := markdown.AsString(rule)
		Expect(err).ToNot(HaveOccurred())

		// Then the default marker is written
		Expect(v).To(Equal("---"))
	})
})
