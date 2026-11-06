package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq passes macro arguments to queries and renderers as they are
// written, with their quotes and spaces. A saved macro must keep the text
// that the user wrote.
var _ = Describe("Macro arguments in saved text", func() {
	keepsMacro := func(name string, input string) {
		It(name, func() {
			// Given a block with a macro
			// When it is parsed and written
			// Then the file text is unchanged
			Expect(parseAndOutput(input)).To(Equal(input))
		})
	}

	Describe("quotes", func() {
		keepsMacro("keeps the quotes of a full text query", `- {{query "full text"}}`)

		It("keeps the quotes in a macro argument when parsed", func() {
			// Given a macro with a quoted argument
			block, err := markdown.ParseString(`- {{foo "x"}}`)
			Expect(err).ToNot(HaveOccurred())

			// When it is parsed
			macro := block.Children().FindDeep(content.IsOfType[*content.Macro]())
			Expect(macro).ToNot(BeNil())

			// Then the argument keeps its quotes, as Logseq gives it to renderers
			Expect(macro.(*content.Macro).Arguments).To(Equal([]string{`"x"`}))
		})
	})

	Describe("spacing", func() {
		keepsMacro("keeps arguments without a space after the comma", "- {{foo a,b}}")
		keepsMacro("keeps a comma in a video URL", "- {{video https://www.youtube.com/watch?v=abc&t=1,2}}")
	})

	Describe("embeds", func() {
		keepsMacro("keeps a second argument of a page embed", "- {{embed [[x]], y}}")
		keepsMacro("keeps the spaces around the target of a page embed", "- {{embed  [[x]] }}")
	})
})
