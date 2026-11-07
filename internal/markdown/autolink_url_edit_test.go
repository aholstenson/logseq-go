package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A bare URL such as `http://e.com` ends before punctuation at its end.
// When a program sets the URL of a bare autolink to a URL that ends in
// punctuation, the URL must be written in angle brackets.
var _ = Describe("Changed URL of a bare autolink", func() {
	It("writes a URL that ends with a dot in angle brackets", func() {
		// Given a block with a bare autolink
		root, err := markdown.ParseString("- http://e.com x")
		Expect(err).ToNot(HaveOccurred())
		link := root.Children().FindDeep(content.IsOfType[*content.AutoLink]())
		Expect(link).ToNot(BeNil())

		// When the URL is set to `http://e.com/x.` and the block is written
		link.(*content.AutoLink).WithURL("http://e.com/x.")
		output, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())

		// Then the URL is in angle brackets
		Expect(output).To(Equal("- <http://e.com/x.> x"))
	})
})
