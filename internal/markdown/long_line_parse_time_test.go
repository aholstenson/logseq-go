package markdown_test

import (
	"strings"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Pages with very long lines (pasted articles, logs, generated text) are
// read when a graph is opened and every time the file changes. The time to
// read a line must grow with its length, not with the square of its length.
var _ = Describe("Parse time of long lines", func() {
	It("reads a 200 KB line of words in less than 2 seconds", func() {
		// Given a block with one line of 40 000 words (200 KB)
		input := "- " + strings.Repeat("word ", 40000)

		// When it is parsed
		start := time.Now()
		_, err := markdown.ParseString(input)
		elapsed := time.Since(start)

		// Then it takes less than 2 seconds
		Expect(err).ToNot(HaveOccurred())
		Expect(elapsed).To(BeNumerically("<", 2*time.Second))
	})

	DescribeTable("reads and writes a 400 KB line with many starts of markup in less than 2 seconds",
		func(unit string) {
			// Given a block with one line of 400 KB that repeats the unit
			input := "- " + strings.Repeat(unit, 400000/len(unit))

			// When it is parsed and written
			start := time.Now()
			root, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())
			parsed := time.Since(start)

			start = time.Now()
			_, err = markdown.AsString(root)
			Expect(err).ToNot(HaveOccurred())
			written := time.Since(start)

			// Then each takes less than 2 seconds
			Expect(parsed).To(BeNumerically("<", 2*time.Second))
			Expect(written).To(BeNumerically("<", 2*time.Second))
		},
		Entry("tags", "#t "),
		Entry("emphasis markers", "*a "),
		Entry("strikethrough markers", "~~a "),
		Entry("dollar signs", "$a "),
		Entry("macros that are not closed", "{{a "),
		Entry("formulas with backslashes that are not closed", "\\( "),
		Entry("page links that are not closed", "[[a "),
		Entry("tags with brackets that are not closed", "#[[a "),
		Entry("links that are not closed", "[a](b "),
		Entry("closing brackets before a parenthesis", "](a "),
		Entry("angle brackets", "<a "),
	)

	It("does not make links from bare www addresses and e-mail addresses", func() {
		// Given a block with a www address and an e-mail address
		block, err := markdown.ParseString("- see www.example.com and a@b.com")
		Expect(err).ToNot(HaveOccurred())

		// When it is parsed
		links := block.Children().FilterDeep(content.IsEither(
			content.IsOfType[*content.Link](),
			content.IsOfType[*content.AutoLink](),
		))

		// Then neither becomes a link
		Expect(links).To(BeEmpty())
	})
})
