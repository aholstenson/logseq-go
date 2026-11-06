package content_test

import (
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NestedPageRef", func() {
	// nestedRefs gets the references in the paragraph that are links inside
	// the title of another reference.
	nestedRefs := func(paragraph *content.Paragraph) []*content.NestedPageRef {
		refs := []*content.NestedPageRef{}
		for _, ref := range paragraph.Children().PageReferences() {
			if nested, ok := ref.(*content.NestedPageRef); ok {
				refs = append(refs, nested)
			}
		}
		return refs
	}

	It("refers to each page link in the title of a page link", func() {
		link := content.NewPageLink("a [[b]] c [[d]]")
		refs := nestedRefs(content.NewParagraph(link))

		Expect(refs).To(HaveLen(2))
		Expect(refs[0].GetTo()).To(Equal("b"))
		Expect(refs[1].GetTo()).To(Equal("d"))
		Expect(refs[0].Outer()).To(BeIdenticalTo(link))
	})

	It("changes the title of the outer link when it is pointed at another page", func() {
		// Given a page link with the page b linked twice in its title
		link := content.NewPageLink("a [[b]] c [[b]]")
		refs := nestedRefs(content.NewParagraph(link))

		// When both are pointed at a page with a longer title
		for _, ref := range refs {
			ref.SetTo("bb")
		}

		// Then both links in the title point at the new page
		Expect(link.To).To(Equal("a [[bb]] c [[bb]]"))
	})

	It("refers to the page links in the title of a tag", func() {
		refs := nestedRefs(content.NewParagraph(content.NewHashtag("a [[b]]")))

		Expect(refs).To(HaveLen(1))
		Expect(refs[0].GetTo()).To(Equal("b"))
	})

	It("does not refer to a page link that is not closed", func() {
		Expect(nestedRefs(content.NewParagraph(content.NewPageLink("a [[b")))).To(BeEmpty())
	})
})
