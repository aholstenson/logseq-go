package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Two emphasis nodes can be next to each other when they use different
// markers, as in `*a*_b_`. Saving the block does not add a space between
// them.
var _ = Describe("Parsing then outputting emphasis nodes next to each other", func() {
	// Given italic text with `*` next to italic text with `_`, When
	// written, Then no space is added.
	FullyEqual("Italic with a star next to italic with an underscore", "- *a*_b_")

	// Given two emphasis nodes made with the API, When written and read
	// again, Then there are still two emphasis nodes. This works now and
	// must keep working.
	It("keeps two emphasis nodes made with the API", func() {
		v, err := markdown.AsString(content.NewParagraph(
			content.NewEmphasis(content.NewText("a")),
			content.NewEmphasis(content.NewText("b")),
		))
		Expect(err).ToNot(HaveOccurred())

		again, err := markdown.ParseString(v)
		Expect(err).ToNot(HaveOccurred())
		Expect(again.Children().FilterDeep(content.IsOfType[*content.Emphasis]())).To(HaveLen(2), "text: %q", v)
	})
})
