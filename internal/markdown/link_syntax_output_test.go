package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A saved link must keep the syntax that the user wrote, and it must still
// be a link when the page is read again.
var _ = Describe("Link syntax in saved blocks", func() {
	It("keeps double quotes around a link title", func() {
		// Given a link with a title in double quotes, When it is written
		input := `- [x](http://a.com "title")`

		// Then the file text is unchanged
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("keeps the angle brackets of an autolink", func() {
		// Given an autolink in angle brackets, When it is written
		input := "- <http://b.com>"

		// Then the file text is unchanged
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("keeps a link with spaces in the destination as a link after it is saved", func() {
		// Given a link with a destination in angle brackets that has spaces
		block, err := markdown.ParseString("- [link](<url with spaces>)")
		Expect(err).ToNot(HaveOccurred())

		// When it is written and read again
		written, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())
		again, err := markdown.ParseString(written)
		Expect(err).ToNot(HaveOccurred())

		// Then it is still a link to `url with spaces`
		link := again.Children().FindDeep(content.IsOfType[*content.Link]())
		Expect(link).ToNot(BeNil(), "written text: %q", written)
		Expect(link.(*content.Link).URL).To(Equal("url with spaces"))
	})

	keepsLink := func(name string, input string) {
		It(name, func() {
			// Given a block with a link in this syntax
			// When it is parsed and written
			// Then the file text is unchanged
			Expect(parseAndOutput(input)).To(Equal(input))
		})
	}

	keepsLink("keeps single quotes around a link title", `- [x](http://a.com 'title')`)
	keepsLink("keeps parentheses around a link title", `- [x](http://a.com (title))`)
	keepsLink("keeps parentheses inside a URL", `- [link](url(with)parens)`)
	keepsLink("keeps an escaped parenthesis inside a URL", `- [x](a\)b)`)
	keepsLink("keeps the spaces around a link title", `- [x](http://a.com  "t" )`)
	keepsLink("keeps empty angle brackets", `- [x](<>)`)
	keepsLink("keeps double quotes around an image title", `- ![a](../assets/x.png "t")`)
	keepsLink("keeps an image title and image size metadata", `- ![a](x.png "t"){:width 1}`)
	keepsLink("keeps a link in a quote", "- > [q](<a b> \"t\")")
	keepsLink("keeps an autolink in angle brackets inside text", "- see <ftp://x.org/a> now")
	keepsLink("keeps an email autolink", "- <a@b.com>")
	keepsLink("keeps a URL without angle brackets", "- http://b.com")

	It("writes a link with a changed URL in the default syntax", func() {
		// Given a link that was read with a title in double quotes
		block, err := markdown.ParseString(`- [x](http://a.com "title")`)
		Expect(err).ToNot(HaveOccurred())
		link := block.Children().FindDeep(content.IsOfType[*content.Link]()).(*content.Link)

		// When its URL is changed and it is written
		link.URL = "http://b.com"
		written, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())

		// Then the new URL is written, with the title in double quotes
		Expect(written).To(Equal(`- [x](http://b.com "title")`))
	})

	It("writes a link with a changed title in the default syntax", func() {
		// Given a link that was read with a title in double quotes
		block, err := markdown.ParseString(`- [x](http://a.com "title")`)
		Expect(err).ToNot(HaveOccurred())
		link := block.Children().FindDeep(content.IsOfType[*content.Link]()).(*content.Link)

		// When its title is changed and it is written
		link.Title = "new"
		written, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())

		// Then the new title is written
		Expect(written).To(Equal(`- [x](http://a.com "new")`))
	})

	It("writes a URL with spaces from the API in angle brackets", func() {
		// Given a link made with the API to a URL with spaces
		paragraph := content.NewParagraph(content.NewLink("a b", content.NewText("x")))

		// When it is written and read again
		written, err := markdown.AsString(paragraph)
		Expect(err).ToNot(HaveOccurred())
		again, err := markdown.ParseString(written)
		Expect(err).ToNot(HaveOccurred())

		// Then the URL is in angle brackets and it is still a link to the URL
		Expect(written).To(Equal("[x](<a b>)"))
		link := again.Children().FindDeep(content.IsOfType[*content.Link]())
		Expect(link).ToNot(BeNil())
		Expect(link.(*content.Link).URL).To(Equal("a b"))
	})

	It("reads if an autolink is in angle brackets", func() {
		// Given an autolink in angle brackets and a URL without them
		block, err := markdown.ParseString("<http://a.com> http://b.com")
		Expect(err).ToNot(HaveOccurred())

		// When it is parsed
		links := block.Children().FilterDeep(content.IsOfType[*content.AutoLink]())

		// Then only the first is marked as in angle brackets
		Expect(links).To(HaveLen(2))
		Expect(links[0].(*content.AutoLink).Bracketed).To(BeTrue())
		Expect(links[1].(*content.AutoLink).Bracketed).To(BeFalse())
	})
})
