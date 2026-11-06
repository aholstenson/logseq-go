package markdown_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq reads `\[ ... \]` as a display math formula. So square brackets in
// text that the user typed must be saved as they are, and not escaped as
// `\[` and `\]`.
var _ = Describe("Square brackets in saved text", func() {
	keepsBrackets := func(name string, input string) {
		It(name, func() {
			// Given a page with square brackets that are not a link
			// When it is parsed and written
			written := parseAndOutput(input)

			// Then the file text is unchanged and has no `\[`
			Expect(strings.Contains(written, `\[`)).To(BeFalse(), "written text: %q", written)
			Expect(written).To(Equal(input))
		})
	}

	Describe("plain text", func() {
		keepsBrackets("keeps a word in brackets", "- see [note] here")
		keepsBrackets("keeps numbered citations", "- see [1] and [2]")
		keepsBrackets("keeps an array index", "- arr[0] = 1")
		keepsBrackets("keeps an empty checkbox", "- [ ] checkbox")
		keepsBrackets("keeps an inactive timestamp", "- [2024-01-01 Mon 10:00]")
		keepsBrackets("keeps hiccup", `- [:div "x"]`)
	})

	Describe("links that Logseq reads but we do not", func() {
		keepsBrackets("keeps a label link to a page name with a space", "- [lbl](Page name)")
		keepsBrackets("keeps an Obsidian style embed of a page", "- ![[My Page]]")
	})

	Describe("page links", func() {
		It("keeps an Org style file link with a label", func() {
			// Given `[[file:./a.md][A]]`, When it is written
			input := "- [[file:./a.md][A]]"

			// Then the file text is unchanged
			Expect(parseAndOutput(input)).To(Equal(input))
		})
	})

	Describe("property values", func() {
		keepsBrackets("keeps brackets in a page title property", "title:: Notes [draft]\n\n- a")
		keepsBrackets("keeps a single opening bracket in a block property", "- a\n  foo:: [x")
	})
})
