package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// hashtagOf gets the first tag of a text.
func hashtagOf(text string) *content.Hashtag {
	root, err := markdown.ParseString(text)
	Expect(err).ToNot(HaveOccurred())
	tag := root.Children().FindDeep(content.IsOfType[*content.Hashtag]())
	Expect(tag).ToNot(BeNil(), "text: %q", text)
	return tag.(*content.Hashtag)
}

// A tag right before a `]`, such as `#a` in "[see #a]", is the tag `a]`
// for Logseq. Saving the block keeps the tag name, and a second save gives
// the same text, so that backslashes are not added on each save.
var _ = Describe("Parsing then outputting tags with a closing bracket in the name", func() {
	// Given a tag right before a closing bracket, When written, read and
	// written again, Then both outputs are the input and the tag is `a]`.
	It("keeps a tag before a closing bracket over two saves", func() {
		input := "- [see #a]"
		first := parseAndOutput(input)
		Expect(first).To(Equal(input))
		second := parseAndOutput(first)
		Expect(second).To(Equal(input))
		Expect(hashtagOf(second).To).To(Equal("a]"))
	})

	// Given a tag whose name is changed to `a]`, When written and read
	// again, Then the name is `a]` and a second save gives the same text.
	It("keeps a tag name with a closing bracket that is set with the API", func() {
		root, err := markdown.ParseString("- #t x")
		Expect(err).ToNot(HaveOccurred())
		tag := root.Children().FindDeep(content.IsOfType[*content.Hashtag]())
		Expect(tag).ToNot(BeNil())
		tag.(*content.Hashtag).SetTo("a]")

		first, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())
		Expect(hashtagOf(first).To).To(Equal("a]"), "text: %q", first)
		Expect(parseAndOutput(first)).To(Equal(first))
	})

	// Given a bracketed tag with a space, When written, Then the text is
	// the same. This works now and must keep working.
	FullyEqual("Bracketed tag with a space", "- #[[a b]]")
})
