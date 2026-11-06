package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	"github.com/aholstenson/logseq-go/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Emphasis characters that cannot open or close emphasis must be saved as
// the user typed them. Logseq shows the raw file text in edit mode, so an
// added backslash is visible there, and property values keep it.
var _ = Describe("Emphasis characters in saved text", func() {
	keepsText := func(name string, input string) {
		It(name, func() {
			// Given a page with emphasis characters that are not emphasis
			// When it is parsed and written
			// Then the file text is unchanged
			Expect(parseAndOutput(input)).To(Equal(input))
		})
	}

	Describe("plain text", func() {
		keepsText("keeps underscores inside a word", "- snake_case_name")
		keepsText("keeps an underscore between CJK characters", "- 日本語_テキスト")
		keepsText("keeps a file name and a multiplication", "- file_name.md and 2 * 3")
		keepsText("keeps a single asterisk between spaces", "- a * b")
		keepsText("keeps a lone pair of tildes", "- ~~ alone")
		keepsText("keeps a lone pair of carets", "- ^^ alone")
	})

	Describe("markers that nothing can pair with", func() {
		keepsText("keeps underscores that open but are not closed", "- __a__b")
		keepsText("keeps underscores that close but are not opened", "- a__b__")
		keepsText("keeps asterisks that open at the end of the text", "- **a")
		keepsText("keeps asterisks that close at the end of the text", "- a**")
		keepsText("keeps tildes that open at the end of the text", "- ~~a")
		keepsText("keeps carets that open at the end of the text", "- a ^^b")
		keepsText("keeps escaped markers that a later line can close", "- \\_\\_a\n  b\\_\\_")
		keepsText("keeps escaped markers that a page link is between", "- \\_\\_a [[p]] b\\_\\_")
	})

	Describe("property values", func() {
		It("keeps underscores in an asset path property", func() {
			// Given a block property with an asset path that has underscores
			input := "- a\n  file:: ../assets/my_doc_v2.pdf"

			// When it is parsed
			block, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			// Then the property value is the path
			value := block.Blocks()[0].Properties().Get("file")
			text := value.FindDeep(content.IsOfType[*content.Text]())
			Expect(text).ToNot(BeNil())
			Expect(text.(*content.Text).Value).To(Equal("../assets/my_doc_v2.pdf"))

			// And when it is written, Then the file text is unchanged
			written, err := markdown.AsString(block)
			Expect(err).ToNot(HaveOccurred())
			Expect(written).To(Equal(input))
		})

		keepsText("keeps underscores in a property value", "- a\n  id_ref:: a_b_c")
	})

	Describe("text made with the API", func() {
		It("still escapes text that would read back as emphasis", func() {
			// Given a Text node with the value `*x*`
			block := content.NewBlock(content.NewParagraph(content.NewText("*x*")))

			// When it is written and read again
			written, err := markdown.AsString(block)
			Expect(err).ToNot(HaveOccurred())
			again, err := markdown.ParseString(written)
			Expect(err).ToNot(HaveOccurred())

			// Then it is the text `*x*` and not emphasis
			Expect(again).To(tests.EqualNode(block))
		})

		It("escapes markers that a node after the text can close", func() {
			// Given a text that opens with `__` and a text after a page link that
			// closes with `__`
			block := content.NewBlock(content.NewParagraph(
				content.NewText("__a "),
				content.NewPageLink("p"),
				content.NewText(" b__"),
			))

			// When it is written and read again
			written, err := markdown.AsString(block)
			Expect(err).ToNot(HaveOccurred())
			again, err := markdown.ParseString(written)
			Expect(err).ToNot(HaveOccurred())

			// Then both texts keep their markers and no strong text is made
			Expect(written).To(Equal("\\_\\_a [[p]] b\\_\\_"))
			Expect(again).To(tests.EqualNode(block))
		})

		It("escapes markers when the text around them is not known", func() {
			// Given a text that opens with `__` that nothing in it closes
			// When it is escaped on its own
			// Then the markers are escaped, as text after it could close them
			Expect(markdown.EscapePotentialMarkdown("__a__b")).To(Equal("\\_\\_a__b"))
		})
	})
})
