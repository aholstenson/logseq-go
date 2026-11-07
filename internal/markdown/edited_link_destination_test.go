package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// When a program changes a link or an image, or makes a new one, the
// destination must be written in a form that Logseq reads as the same URL
// and title. Logseq reads a title only in double quotes, and it reads a
// backslash before a parenthesis in a URL as part of the URL.
var _ = Describe("Destinations of changed links and images", func() {
	parse := func(input string) *content.Block {
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		return root
	}

	write := func(node content.Node) string {
		v, err := markdown.AsString(node)
		Expect(err).ToNot(HaveOccurred())
		return v
	}

	firstLink := func(root *content.Block) *content.Link {
		link := root.Children().FindDeep(content.IsOfType[*content.Link]())
		Expect(link).ToNot(BeNil())
		return link.(*content.Link)
	}

	It("writes the title in double quotes when the URL of a link is changed", func() {
		// Given a link with a title in double quotes
		root := parse(`- [x](http://a.com "title")`)

		// When the URL of the link is changed and the block is written
		firstLink(root).WithURL("http://b.com")

		// Then the title is still in double quotes
		Expect(write(root)).To(Equal(`- [x](http://b.com "title")`))
	})

	It("writes a new URL with parentheses without backslashes", func() {
		// Given a link without a title
		root := parse("- [x](http://a.com)")

		// When the URL is changed to a URL with balanced parentheses
		firstLink(root).WithURL("https://e.com/x(y)")
		output := write(root)

		// Then the URL is written in a form that Logseq reads as the same URL
		Expect(output).To(BeElementOf(
			"- [x](https://e.com/x(y))",
			"- [x](<https://e.com/x(y)>)",
		))

		// And no backslash is put before a parenthesis
		Expect(output).ToNot(ContainSubstring(`\(`))
		Expect(output).ToNot(ContainSubstring(`\)`))
	})

	It("writes the title of a new image in double quotes", func() {
		// Given an image with a title that is made with the API
		paragraph := content.NewParagraph(
			content.NewImage("p.png", content.NewText("a")).WithTitle("t"),
		)

		// When it is written, Then the title is in double quotes
		Expect(write(paragraph)).To(Equal(`![a](p.png "t")`))
	})

	It("keeps single quotes around the title of a link that is not changed", func() {
		// Given a link with a title in single quotes
		input := "- [x](http://a.com 'title')"

		// When it is read and written without a change, Then the text is the same
		Expect(write(parse(input))).To(Equal(input))
	})
})
