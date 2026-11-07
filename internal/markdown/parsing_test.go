package markdown_test

import (
	"strings"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	"github.com/aholstenson/logseq-go/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parsing", func() {
	Describe("Basic content", func() {
		It("can parse text", func() {
			block, err := markdown.ParseString("This is some basic text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is some basic text"),
				),
			)))
		})

		It("can parse text with escaped characters", func() {
			block, err := markdown.ParseString("This is some basic text with \\*escaped\\* characters")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is some basic text with *escaped* characters"),
				),
			)))
		})

		It("can parse text with newline", func() {
			block, err := markdown.ParseString("This is some basic text\nwith a newline")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is some basic text").WithSoftLineBreak(),
					content.NewText("with a newline"),
				),
			)))
		})

		It("can parse text with hard newline", func() {
			block, err := markdown.ParseString("This is some basic text  \nwith a newline")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is some basic text").WithHardLineBreak(),
					content.NewText("with a newline"),
				),
			)))
		})

		It("can parse heading", func() {
			block, err := markdown.ParseString("# Headline")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewHeading(1, content.NewText("Headline")),
			)))
		})

		It("can parse heading level 2", func() {
			block, err := markdown.ParseString("## Headline")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewHeading(2, content.NewText("Headline")),
			)))
		})

		It("can parse heading and text", func() {
			block, err := markdown.ParseString("# Headline\n\nThis is some basic text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewHeading(1, content.NewText("Headline")),
				content.NewParagraph(
					content.NewText("This is some basic text"),
				),
			)))
		})

		It("can parse heading and text with only single newline", func() {
			block, err := markdown.ParseString("# Headline\nThis is some basic text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewHeading(1, content.NewText("Headline")),
				content.NewParagraph(
					content.NewText("This is some basic text"),
				).WithPreviousLineType(content.PreviousLineTypeNonBlank),
			)))
		})

		It("can parse soft line breaks", func() {
			block, err := markdown.ParseString("This is some basic text\nwith a soft line break")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is some basic text").WithSoftLineBreak(),
					content.NewText("with a soft line break"),
				),
			)))
		})
	})

	Describe("Inline formatting", func() {
		It("can parse emphasis", func() {
			block, err := markdown.ParseString("This is *emphasized* text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is "),
					content.NewEmphasis(content.NewText("emphasized")),
					content.NewText(" text"),
				),
			)))
		})

		It("can parse strong", func() {
			block, err := markdown.ParseString("This is **strong** text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is "),
					content.NewStrong(content.NewText("strong")),
					content.NewText(" text"),
				),
			)))
		})

		It("can parse strong with a space before the closing marker", func() {
			block, err := markdown.ParseString("**Card from [[A]] and [[B]] **")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewStrong(
						content.NewText("Card from "),
						content.NewPageLink("A"),
						content.NewText(" and "),
						content.NewPageLink("B"),
						content.NewText(" "),
					),
				),
			)))
		})

		It("can parse strong parts that follow each other", func() {
			block, err := markdown.ParseString("**a** **b**")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewStrong(content.NewText("a")),
					content.NewText(" "),
					content.NewStrong(content.NewText("b")),
				),
			)))
		})

		DescribeTable("can parse a space before a closing marker at the end of the line",
			func(input string, expected content.InlineNode) {
				block, err := markdown.ParseString(input)
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(expected),
				)))
			},
			Entry("italic with *", "*a *", content.NewEmphasis(content.NewText("a "))),
			Entry("italic with _", "_a _", content.NewEmphasis(content.NewText("a ")).WithUnderscore(true)),
			Entry("strong with __", "__a __", content.NewStrong(content.NewText("a ")).WithUnderscore(true)),
			Entry("strikethrough", "~~a ~~", content.NewStrikethrough(content.NewText("a "))),
			Entry("highlight", "^^a ^^", content.NewHighlight(content.NewText("a "))),
		)

		It("can parse a space before a closing marker before a newline", func() {
			block, err := markdown.ParseString("**a **\nb")
			Expect(err).ToNot(HaveOccurred())

			paragraph := block.FirstChild().(*content.Paragraph)
			Expect(paragraph.FirstChild()).To(BeAssignableToTypeOf(&content.Strong{}))
		})

		It("does not parse strong from a space before a marker in the middle of a line", func() {
			block, err := markdown.ParseString("**a ** b")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("**a ** b"),
				),
			)))
		})

		It("does not parse strong from markers surrounded by spaces", func() {
			block, err := markdown.ParseString("a ** b ** c")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("a ** b ** c"),
				),
			)))
		})

		It("can parse emphasis and strong", func() {
			block, err := markdown.ParseString("This is ***strong and emphasized*** text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is "),
					content.NewEmphasis(
						content.NewStrong(
							content.NewText("strong and emphasized"),
						),
					),
					content.NewText(" text"),
				),
			)))
		})

		It("can parse strikethrough", func() {
			block, err := markdown.ParseString("This is ~~strikethrough~~ text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is "),
					content.NewStrikethrough(content.NewText("strikethrough")),
					content.NewText(" text"),
				),
			)))
		})

		It("can parse strikethrough and strong", func() {
			block, err := markdown.ParseString("This is ~~**strikethrough and strong**~~ text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is "),
					content.NewStrikethrough(
						content.NewStrong(
							content.NewText("strikethrough and strong"),
						),
					),
					content.NewText(" text"),
				),
			)))
		})

		It("can parse strikethrough with ~ inline", func() {
			block, err := markdown.ParseString("This is ~~strike~through~~ text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is "),
					content.NewStrikethrough(content.NewText("strike~through")),
					content.NewText(" text"),
				),
			)))
		})

		It("can parse strikethrough with ~~ inline", func() {
			block, err := markdown.ParseString("This is ~~strike~~through~~ text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is "),
					content.NewStrikethrough(content.NewText("strike")),
					content.NewText("through~~ text"),
				),
			)))
		})

		It("can parse strikethrough with escaped ~", func() {
			block, err := markdown.ParseString("This is ~~strike\\~~through~~ text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is "),
					content.NewStrikethrough(content.NewText("strike~~through")),
					content.NewText(" text"),
				),
			)))
		})

		It("can parse highlight", func() {
			block, err := markdown.ParseString("This is ^^highlighted^^ text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is "),
					content.NewHighlight(content.NewText("highlighted")),
					content.NewText(" text"),
				),
			)))
		})

		It("can parse highlight and strong", func() {
			block, err := markdown.ParseString("This is ^^**highlighted and strong**^^ text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is "),
					content.NewHighlight(
						content.NewStrong(
							content.NewText("highlighted and strong"),
						),
					),
					content.NewText(" text"),
				),
			)))
		})

		It("can parse highlight with ^ inline", func() {
			block, err := markdown.ParseString("This is ^^high^lighted^^ text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is "),
					content.NewHighlight(content.NewText("high^lighted")),
					content.NewText(" text"),
				),
			)))
		})

		It("keeps a single ^ as text", func() {
			block, err := markdown.ParseString("This is 2^10 text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is 2^10 text"),
				),
			)))
		})

		It("keeps an unmatched ^^ as text", func() {
			block, err := markdown.ParseString("This is ^^ text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is ^^ text"),
				),
			)))
		})

		It("can parse code", func() {
			block, err := markdown.ParseString("This is `code` text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is "),
					content.NewCodeSpan("code"),
					content.NewText(" text"),
				),
			)))
		})

		It("can parse code with escaped characters", func() {
			block, err := markdown.ParseString("`code\\`")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewCodeSpan("code\\"),
				),
			)))
		})

		It("can parse code with double backticks", func() {
			block, err := markdown.ParseString("``code``")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewCodeSpan("code"),
				),
			)))
		})

		It("can parse code with double backticks and inline backtick", func() {
			block, err := markdown.ParseString("``co`de``")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewCodeSpan("co`de"),
				),
			)))
		})

		It("can parse code with triple backticks", func() {
			block, err := markdown.ParseString("```code```")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewCodeSpan("code"),
				),
			)))
		})
	})

	Describe("Links", func() {
		It("can parse link", func() {
			block, err := markdown.ParseString("[This is a link](https://example.com)")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewLink(
						"https://example.com",
						content.NewText("This is a link"),
					),
				),
			)))
		})

		It("does not parse a tag in the text of a link", func() {
			block, err := markdown.ParseString("**08:54** #Article [Smaller Teams:#](https://a.com/) #Readwise")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewStrong(content.NewText("08:54")),
					content.NewText(" "),
					content.NewHashtag("Article"),
					content.NewText(" "),
					content.NewLink(
						"https://a.com/",
						content.NewText("Smaller Teams:#"),
					),
					content.NewText(" "),
					content.NewHashtag("Readwise"),
				),
			)))
		})

		It("does not parse a tag with a name in the text of a link", func() {
			block, err := markdown.ParseString("[a #tag [b] c](https://a.com/)")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewLink(
						"https://a.com/",
						content.NewText("a #tag [b] c"),
					),
				),
			)))
		})

		It("parses a tag in brackets that are not a link", func() {
			block, err := markdown.ParseString("[a #tag] (b)")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("[a "),
					content.NewHashtag("tag]"),
					content.NewText(" (b)"),
				),
			)))
		})

		It("can parse link with escaped characters", func() {
			block, err := markdown.ParseString("[This is a link](https://example.com\\)\\*\\[\\])")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewLink(
						"https://example.com)*[]",
						content.NewText("This is a link"),
					),
				),
			)))
		})

		It("can parse link with title", func() {
			block, err := markdown.ParseString("[This is a link](https://example.com 'Title')")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewLink(
						"https://example.com",
						content.NewText("This is a link"),
					).WithTitle("Title"),
				),
			)))
		})

		It("can parse link with title and newlines", func() {
			block, err := markdown.ParseString("[This is a link](https://example.com 'Title\nwith newlines')")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewLink(
						"https://example.com",
						content.NewText("This is a link"),
					).WithTitle("Title\nwith newlines"),
				),
			)))
		})

		It("can parse link with escaped title", func() {
			block, err := markdown.ParseString("[This is a link](https://example.com 'Title\\'s')")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewLink(
						"https://example.com",
						content.NewText("This is a link"),
					).WithTitle("Title's"),
				),
			)))
		})

		It("can parse auto link", func() {
			block, err := markdown.ParseString("<https://example.com>")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewAutoLink("https://example.com").WithBracketed(true),
				),
			)))
		})

		It("can parse auto link without brackets", func() {
			block, err := markdown.ParseString("https://example.com")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewAutoLink("https://example.com"),
				),
			)))
		})

		It("link with only www prefix is not an auto link", func() {
			block, err := markdown.ParseString("www.example.com")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("www.example.com"),
				),
			)))
		})

		It("e-mails are not auto linked", func() {
			block, err := markdown.ParseString("test@example.com")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("test@example.com"),
				),
			)))
		})

		It("can parse simple tags", func() {
			block, err := markdown.ParseString("#tag followed by some text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewHashtag("tag"),
					content.NewText(" followed by some text"),
				),
			)))
		})

		It("can parse tags with spaces", func() {
			block, err := markdown.ParseString("#[[tag with spaces]] and some other content")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewHashtag("tag with spaces"),
					content.NewText(" and some other content"),
				),
			)))
		})

		It("can parse tag when end of line", func() {
			block, err := markdown.ParseString("#tag")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewHashtag("tag"),
				),
			)))
		})

		It("does not treat a hash followed by a space as a tag", func() {
			block, err := markdown.ParseString("issue # 5")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("issue # 5"),
				),
			)))
		})

		It("does not treat a hash at the end of a word as a tag", func() {
			block, err := markdown.ParseString("C# is fine")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("C# is fine"),
				),
			)))
		})

		It("does not treat a hash after a word and before punctuation as a tag", func() {
			block, err := markdown.ParseString("Like C#, C and Go")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("Like C#, C and Go"),
				),
			)))
		})

		It("does not treat a hash inside a word as a tag", func() {
			block, err := markdown.ParseString("C#Sharp and https://a.com/#top")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("C#Sharp and "),
					content.NewAutoLink("https://a.com/#top"),
				),
			)))
		})

		It("does not treat a hash after a closing bracket as a tag", func() {
			block, err := markdown.ParseString("(#tag)")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("(#tag)"),
				),
			)))
		})

		It("can parse tag after punctuation", func() {
			block, err := markdown.ParseString("a,#first and \"#second\"")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("a,"),
					content.NewHashtag("first"),
					content.NewText(" and \""),
					content.NewHashtag("second"),
					content.NewText("\""),
				),
			)))
		})

		It("can parse tag after a non-breaking space", func() {
			block, err := markdown.ParseString("a #tag")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("a "),
					content.NewHashtag("tag"),
				),
			)))
		})

		It("does not include trailing punctuation in the tag", func() {
			block, err := markdown.ParseString("#tag,.? and #a.b.c")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewHashtag("tag"),
					content.NewText(",.? and "),
					content.NewHashtag("a.b.c"),
				),
			)))
		})

		It("ends a tag at punctuation in the middle of the name", func() {
			block, err := markdown.ParseString("#a,b #c:d #e\"f #g!h #i?j #k'l")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewHashtag("a"),
					content.NewText(",b "),
					content.NewHashtag("c"),
					content.NewText(":d "),
					content.NewHashtag("e"),
					content.NewText("\"f "),
					content.NewHashtag("g"),
					content.NewText("!h "),
					content.NewHashtag("i"),
					content.NewText("?j "),
					content.NewHashtag("k"),
					content.NewText("'l"),
				),
			)))
		})

		It("keeps a dot or semicolon in the middle of a tag", func() {
			block, err := markdown.ParseString("#a;b #c..d #.e #f.,g")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewHashtag("a;b"),
					content.NewText(" "),
					content.NewHashtag("c..d"),
					content.NewText(" "),
					content.NewHashtag(".e"),
					content.NewText(" "),
					content.NewHashtag("f."),
					content.NewText(",g"),
				),
			)))
		})

		It("keeps brackets in a tag", func() {
			block, err := markdown.ParseString("#a[b]c and #d(e)")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewHashtag("a[b]c"),
					content.NewText(" and "),
					content.NewHashtag("d(e)"),
				),
			)))
		})

		It("does not treat a hash with only punctuation as a tag", func() {
			block, err := markdown.ParseString("x #. y #,a")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("x #. y #,a"),
				),
			)))
		})

		It("starts a new tag at a hash directly after a tag", func() {
			block, err := markdown.ParseString("#a#b")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewHashtag("a"),
					content.NewHashtag("b"),
				),
			)))
		})

		It("ends a tag before a colon in a URL", func() {
			block, err := markdown.ParseString("x #\\](https://a.com/) y")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("x "),
					content.NewHashtag("\\](https"),
					content.NewText("://a.com/) y"),
				),
			)))
		})

		It("starts a tag at the second hash of a double hash", func() {
			block, err := markdown.ParseString("x ##tag")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("x #"),
					content.NewHashtag("tag"),
				),
			)))
		})

		It("does not end a tag inside a multi-byte character", func() {
			// A non-breaking space ends the tag, but only as a whole character
			// and not on the byte of it that looks like one.
			block, err := markdown.ParseString("#tag\u00a0name")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewHashtag("tag"),
					content.NewText("\u00a0name"),
				),
			)))
		})

		It("will skip tag with spaces if end not found", func() {
			block, err := markdown.ParseString("#[[tag with spaces")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("#[[tag with spaces"),
				),
			)))
		})

		It("can parse wiki-style link", func() {
			block, err := markdown.ParseString("[[This is a link]] and this is some text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewPageLink("This is a link"),
					content.NewText(" and this is some text"),
				),
			)))
		})

		It("can parse wiki-style link with ] in it", func() {
			block, err := markdown.ParseString("[[This is ]a link]] and this is some text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewPageLink("This is ]a link"),
					content.NewText(" and this is some text"),
				),
			)))
		})

		It("can parse wiki-style link with escaped ]]", func() {
			block, err := markdown.ParseString("[[This is \\]]a link]] and this is some text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewPageLink("This is ]]a link"),
					content.NewText(" and this is some text"),
				),
			)))
		})

		It("can parse multiple wiki-style links", func() {
			block, err := markdown.ParseString("[[This is a link]] and [[this is another link]]")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewPageLink("This is a link"),
					content.NewText(" and "),
					content.NewPageLink("this is another link"),
				),
			)))
		})

		It("can parse block reference", func() {
			block, err := markdown.ParseString("((0b48a6c6-93ca-4d35-b945-6c59007f7962))")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewBlockRef("0b48a6c6-93ca-4d35-b945-6c59007f7962"),
				),
			)))
		})

		It("can parse page link with a label", func() {
			block, err := markdown.ParseString("See [Goals]([[Workshop/Goals]]) here")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("See "),
					content.NewPageLink("Workshop/Goals", content.NewText("Goals")),
					content.NewText(" here"),
				),
			)))
		})

		It("can parse page link with a label to a page with spaces in the name", func() {
			block, err := markdown.ParseString("See [the goals]([[Workshop Goals]]) here")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("See "),
					content.NewPageLink("Workshop Goals", content.NewText("the goals")),
					content.NewText(" here"),
				),
			)))
		})

		It("can parse page link with a label that has formatting", func() {
			block, err := markdown.ParseString("[**Goals** of it]([[Workshop Goals]])")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewPageLink("Workshop Goals",
						content.NewStrong(content.NewText("Goals")),
						content.NewText(" of it"),
					),
				),
			)))
		})

		It("can parse page link with a label and an escaped ]] in the name", func() {
			block, err := markdown.ParseString("[Goals]([[Work \\]] shop]])")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewPageLink("Work ]] shop", content.NewText("Goals")),
				),
			)))
		})

		It("can parse block reference with a label", func() {
			block, err := markdown.ParseString("See [the block](((0b48a6c6-93ca-4d35-b945-6c59007f7962))) here")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("See "),
					content.NewBlockRef("0b48a6c6-93ca-4d35-b945-6c59007f7962", content.NewText("the block")),
					content.NewText(" here"),
				),
			)))
		})

		It("keeps a link to more than a page link as a link", func() {
			block, err := markdown.ParseString("[Goals]([[Workshop]]/more)")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewLink("[[Workshop]]/more", content.NewText("Goals")),
				),
			)))
		})

		It("keeps text in brackets before a page link as text", func() {
			block, err := markdown.ParseString("[Goals] ([[Workshop Goals]])")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("[Goals] ("),
					content.NewPageLink("Workshop Goals"),
					content.NewText(")"),
				),
			)))
		})
	})

	Describe("Images", func() {
		It("can parse image", func() {
			block, err := markdown.ParseString("![This is an image](https://example.com/image.png)")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewImage(
						"https://example.com/image.png",
						content.NewText("This is an image"),
					),
				),
			)))
		})

		It("can parse image with title", func() {
			block, err := markdown.ParseString("![This is an image](https://example.com/image.png 'Title')")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewImage(
						"https://example.com/image.png",
						content.NewText("This is an image"),
					).WithTitle("Title"),
				),
			)))
		})
	})

	Describe("Macros", func() {
		It("can parse macro with only name", func() {
			block, err := markdown.ParseString("{{macro}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro"),
				),
			)))
		})

		It("macro with one non-quoted argument", func() {
			block, err := markdown.ParseString("{{macro arg}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro", "arg"),
				),
			)))
		})

		It("macro with one non-quoted argument and spaces", func() {
			block, err := markdown.ParseString("{{macro arg with spaces}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro", "arg with spaces"),
				),
			)))
		})

		It("can parse macro with name and two arguments separated by comma", func() {
			block, err := markdown.ParseString("{{macro arg1,arg2}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro", "arg1", "arg2"),
				),
			)))
		})

		It("can parse macro with name and two arguments separated by comma and space", func() {
			block, err := markdown.ParseString("{{macro arg1, arg2}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro", "arg1", "arg2"),
				),
			)))
		})

		It("can parse macro with name and one quoted argument", func() {
			block, err := markdown.ParseString("{{macro \"arg1\", arg2}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro", `"arg1"`, "arg2"),
				),
			)))
		})

		It("can parse macro with name and quoted arguments", func() {
			block, err := markdown.ParseString("{{macro \"arg1\", \"arg2\"}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro", `"arg1"`, `"arg2"`),
				),
			)))
		})

		It("can parse macro with name and one quoted argument containing a comma", func() {
			block, err := markdown.ParseString("{{macro \"arg1,\", arg2}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro", `"arg1,"`, "arg2"),
				),
			)))
		})

		It("fails macro with name and one quoted argument wrongly separated", func() {
			block, err := markdown.ParseString("{{macro \"arg1,\" arg2}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("{{macro \"arg1,\" arg2}}"),
				),
			)))
		})

		It("fails macro with name and one argument followed by a trailing comma", func() {
			block, err := markdown.ParseString("{{macro arg1,}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("{{macro arg1,}}"),
				),
			)))
		})

		It("can parse macro with name and quoted argument with escape", func() {
			block, err := markdown.ParseString("{{macro \"arg1\\\"\"}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro", `"arg1\""`),
				),
			)))
		})

		It("can parse macro with name and quote in the middle of argument", func() {
			block, err := markdown.ParseString("{{macro ar\"g1}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro", "ar\"g1"),
				),
			)))
		})

		It("can parse macro with name and quote in the middle of argument followed by another arg", func() {
			block, err := markdown.ParseString("{{macro ar\"g1, arg2}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro", "ar\"g1", "arg2"),
				),
			)))
		})

		It("can parse macro with name containing dash", func() {
			block, err := markdown.ParseString("{{macro-name}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro-name"),
				),
			)))
		})

		It("can handle macro starting with three curly braces", func() {
			block, err := markdown.ParseString("{{{macro}}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro"),
				),
			)))
		})

		It("can handle macro starting with three curly braces with arguments", func() {
			block, err := markdown.ParseString("{{{macro arg1, arg2}}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro", "arg1", "arg2"),
				),
			)))
		})

		It("empty macro is parsed as text", func() {
			block, err := markdown.ParseString("{{}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("{{}}"),
				),
			)))
		})

		It("macro that does not end is parsed as text", func() {
			block, err := markdown.ParseString("{{macro")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("{{macro"),
				),
			)))
		})

		It("macro without closing curly brace is parsed as text", func() {
			block, err := markdown.ParseString("{{macro}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("{{macro}"),
				),
			)))
		})

		It("macro with three curly braces missing the last closing brace is parsed from the second brace", func() {
			// The third { is not part of a macro, so it is left as text and the
			// rest parses as a regular two brace macro.
			block, err := markdown.ParseString("{{{macro}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("{"),
					content.NewMacro("macro"),
				),
			)))
		})

		It("can handle multiple macros", func() {
			block, err := markdown.ParseString("{{macro1}} {{macro2}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro1"),
					content.NewText(" "),
					content.NewMacro("macro2"),
				),
			)))
		})

		Describe("Query", func() {
			It("can parse query", func() {
				block, err := markdown.ParseString("{{query datalog query}}")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewQuery("datalog query"),
					),
				)))
			})

			It("query without a query string is parsed as macro", func() {
				block, err := markdown.ParseString("{{query}}")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewMacro("query"),
					),
				)))
			})
		})

		Describe("Page embed", func() {
			It("can parse page embed", func() {
				block, err := markdown.ParseString("{{embed [[page]]}}")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewPageEmbed("page"),
					),
				)))
			})

			It("embed without closing square bracket is parsed as macro", func() {
				block, err := markdown.ParseString("{{embed [[page]}}")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewMacro("embed", "[[page]"),
					),
				)))
			})
		})

		Describe("Block embed", func() {
			It("can parse block embed", func() {
				block, err := markdown.ParseString("{{embed ((block))}}")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewBlockEmbed("block"),
					),
				)))
			})

			It("embed without closing parenthesis is parsed as macro", func() {
				block, err := markdown.ParseString("{{embed ((block)}}")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewMacro("embed", "((block)"),
					),
				)))
			})
		})

		Describe("Cloze", func() {
			It("can parse cloze with only answer", func() {
				block, err := markdown.ParseString("{{cloze answer}}")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewCloze("answer"),
					),
				)))
			})

			It("can parse cloze with answer and cue", func() {
				block, err := markdown.ParseString("{{cloze answer \\ cue}}")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewClozeWithCue("answer", "cue"),
					),
				)))
			})
		})
	})

	Describe("Code blocks", func() {
		It("can parse code block", func() {
			block, err := markdown.ParseString("```\nThis is a code block\n```")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewCodeBlock("This is a code block\n"),
			)))
		})

		It("can parse code block with newlines", func() {
			block, err := markdown.ParseString("```\nThis is\na code block\n```")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewCodeBlock("This is\na code block\n"),
			)))
		})

		It("can parse code block with blank lines", func() {
			block, err := markdown.ParseString("```\nThis is\n\na code block\n```")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewCodeBlock("This is\n\na code block\n"),
			)))
		})

		It("can parse code block with language", func() {
			block, err := markdown.ParseString("```go\nThis is a code block\n```")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewCodeBlock("This is a code block\n").WithLanguage("go"),
			)))
		})

		It("reads indented text as a paragraph and not as a code block", func() {
			block, err := markdown.ParseString("    This is indented text\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is indented text"),
				).WithIndentation("    "),
			)))
		})

		It("can parse code block with blank lines in block", func() {
			block, err := markdown.ParseString("- ```\n  This is\n  \n  a code block\n  ```")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewBlock(
					content.NewCodeBlock("This is\n\na code block\n"),
				),
			)))
		})
	})

	Describe("Blockquotes", func() {
		It("can parse blockquote", func() {
			block, err := markdown.ParseString("> This is a blockquote\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewBlockquote(
					content.NewParagraph(
						content.NewText("This is a blockquote"),
					),
				),
			)))
		})

		It("can parse nested blockquote", func() {
			block, err := markdown.ParseString("> > This is a nested blockquote\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewBlockquote(
					content.NewBlockquote(
						content.NewParagraph(
							content.NewText("This is a nested blockquote"),
						),
					),
				),
			)))
		})

		It("can parse blockquote spanning multiple lines", func() {
			block, err := markdown.ParseString("> This is a blockquote\n> with multiple lines\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewBlockquote(
					content.NewParagraph(
						content.NewText("This is a blockquote").WithSoftLineBreak(),
						content.NewText("with multiple lines"),
					),
				),
			)))
		})

		It("can parse blockquote spanning multiple lines without marker", func() {
			block, err := markdown.ParseString("> This is a blockquote\nwith multiple lines\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewBlockquote(
					content.NewParagraph(
						content.NewText("This is a blockquote").WithSoftLineBreak().WithLazyLine(),
						content.NewText("with multiple lines"),
					),
				),
			)))
		})

		It("can parse blockquote with header", func() {
			block, err := markdown.ParseString("> # This is a header\n> And this is a paragraph\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewBlockquote(
					content.NewHeading(1,
						content.NewText("This is a header"),
					),
					content.NewParagraph(
						content.NewText("And this is a paragraph"),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
				),
			)))
		})

		It("can parse blockquote followed by a paragraph", func() {
			block, err := markdown.ParseString("> This is a blockquote\n\nThis is a paragraph")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewBlockquote(
					content.NewParagraph(
						content.NewText("This is a blockquote"),
					),
				),
				content.NewParagraph(
					content.NewText("This is a paragraph"),
				),
			)))
		})

		It("can parse paragraph interrupted by blockquote", func() {
			block, err := markdown.ParseString("This is a paragraph\n> This is a blockquote")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is a paragraph"),
				),
				content.NewBlockquote(
					content.NewParagraph(
						content.NewText("This is a blockquote"),
					),
				).WithPreviousLineType(content.PreviousLineTypeNonBlank),
			)))
		})
	})

	Describe("Lists", func() {
		Describe("Unordered lists", func() {
			Describe("With stars", func() {
				It("can parse", func() {
					block, err := markdown.ParseString("* Item 1\n* Item 2\n")
					Expect(err).ToNot(HaveOccurred())

					Expect(block).To(tests.EqualNode(content.NewBlock(
						content.NewList(
							content.ListTypeUnordered,
							content.NewListItem(
								content.NewParagraph(
									content.NewText("Item 1"),
								),
							),
							content.NewListItem(
								content.NewParagraph(
									content.NewText("Item 2"),
								),
							),
						),
					)))
				})

				It("can parse with multiple lines", func() {
					block, err := markdown.ParseString("* Item 1\n  with multiple lines\n* Item 2\n")
					Expect(err).ToNot(HaveOccurred())

					Expect(block).To(tests.EqualNode(content.NewBlock(
						content.NewList(
							content.ListTypeUnordered,
							content.NewListItem(
								content.NewParagraph(
									content.NewText("Item 1").WithSoftLineBreak(),
									content.NewText("with multiple lines"),
								),
							),
							content.NewListItem(
								content.NewParagraph(
									content.NewText("Item 2"),
								),
							),
						),
					)))
				})

				It("can parse with multiple paragraphs", func() {
					block, err := markdown.ParseString("* Item 1\n\n  with multiple paragraphs\n* Item 2\n")
					Expect(err).ToNot(HaveOccurred())

					Expect(block).To(tests.EqualNode(content.NewBlock(
						content.NewList(
							content.ListTypeUnordered,
							content.NewListItem(
								content.NewParagraph(
									content.NewText("Item 1"),
								),
								content.NewParagraph(
									content.NewText("with multiple paragraphs"),
								),
							),
							content.NewListItem(
								content.NewParagraph(
									content.NewText("Item 2"),
								),
							),
						),
					)))
				})
			})

			Describe("With pluses", func() {
				It("can parse", func() {
					block, err := markdown.ParseString("+ Item 1\n+ Item 2\n")
					Expect(err).ToNot(HaveOccurred())

					Expect(block).To(tests.EqualNode(content.NewBlock(
						content.NewListFromMarker('+',
							content.NewListItem(
								content.NewParagraph(
									content.NewText("Item 1"),
								),
							),
							content.NewListItem(
								content.NewParagraph(
									content.NewText("Item 2"),
								),
							),
						),
					)))
				})
			})

			Describe("With stars and pluses", func() {
				It("can parse", func() {
					block, err := markdown.ParseString("* Item 1\n+ Item 2\n")
					Expect(err).ToNot(HaveOccurred())

					Expect(block).To(tests.EqualNode(content.NewBlock(
						content.NewListFromMarker('*',
							content.NewListItem(
								content.NewParagraph(
									content.NewText("Item 1"),
								),
							),
						),
						content.NewListFromMarker('+',
							content.NewListItem(
								content.NewParagraph(
									content.NewText("Item 2"),
								),
							),
						).WithPreviousLineType(content.PreviousLineTypeNonBlank),
					)))
				})
			})
		})

		Describe("Ordered lists", func() {
			It("can parse", func() {
				block, err := markdown.ParseString("1. Item 1\n2. Item 2\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewList(
						content.ListTypeOrdered,
						content.NewListItem(
							content.NewParagraph(
								content.NewText("Item 1"),
							),
						),
						content.NewListItem(
							content.NewParagraph(
								content.NewText("Item 2"),
							),
						),
					),
				)))
			})

			It("can parse with multiple lines", func() {
				block, err := markdown.ParseString("1. Item 1\n   with multiple lines\n2. Item 2\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewList(
						content.ListTypeOrdered,
						content.NewListItem(
							content.NewParagraph(
								content.NewText("Item 1").WithSoftLineBreak(),
								content.NewText("with multiple lines"),
							),
						),
						content.NewListItem(
							content.NewParagraph(
								content.NewText("Item 2"),
							),
						),
					),
				)))
			})

			It("can parse with multiple paragraphs", func() {
				block, err := markdown.ParseString("1. Item 1\n\n   with multiple paragraphs\n2. Item 2\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewList(
						content.ListTypeOrdered,
						content.NewListItem(
							content.NewParagraph(
								content.NewText("Item 1"),
							),
							content.NewParagraph(
								content.NewText("with multiple paragraphs"),
							),
						),
						content.NewListItem(
							content.NewParagraph(
								content.NewText("Item 2"),
							),
						),
					),
				)))
			})
		})

		Describe("Nested lists", func() {
			It("can parse star within plus", func() {
				block, err := markdown.ParseString("+ Item 1\n  * Item 2\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewListFromMarker('+',
						content.NewListItem(
							content.NewParagraph(
								content.NewText("Item 1"),
							),
							content.NewListFromMarker('*',
								content.NewListItem(
									content.NewParagraph(
										content.NewText("Item 2"),
									),
								),
							).WithPreviousLineType(content.PreviousLineTypeNonBlank),
						),
					),
				)))
			})

			It("can parse star within plus without text", func() {
				block, err := markdown.ParseString("*\n\t* Item 1.1\n\t* Item 2\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewList(
						content.ListTypeUnordered,
						content.NewListItem(
							content.NewList(
								content.ListTypeUnordered,
								content.NewListItem(
									content.NewParagraph(
										content.NewText("Item 1.1"),
									),
								),
								content.NewListItem(
									content.NewParagraph(
										content.NewText("Item 2"),
									),
								),
							),
						),
					),
				)))
			})

			It("can parse star within plus without text", func() {
				block, err := markdown.ParseString("*\n    * Item 1.1\n    * Item 2\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewList(
						content.ListTypeUnordered,
						content.NewListItem(
							content.NewList(
								content.ListTypeUnordered,
								content.NewListItem(
									content.NewParagraph(
										content.NewText("Item 1.1"),
									),
								),
								content.NewListItem(
									content.NewParagraph(
										content.NewText("Item 2"),
									),
								),
							),
						),
					),
				)))
			})
		})
	})

	Describe("Footnotes", func() {
		It("can parse a footnote reference", func() {
			block, err := markdown.ParseString("Text with a footnote[^1] in it")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("Text with a footnote"),
					content.NewFootnoteRef("1"),
					content.NewText(" in it"),
				),
			)))
		})

		It("can parse a footnote reference with a named label", func() {
			block, err := markdown.ParseString("Text[^note]")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("Text"),
					content.NewFootnoteRef("note"),
				),
			)))
		})

		It("can parse a footnote definition", func() {
			block, err := markdown.ParseString("[^1]: The footnote")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewFootnoteDefinition("1",
					content.NewText("The footnote"),
				),
			)))
		})

		It("can parse a footnote definition with formatting", func() {
			block, err := markdown.ParseString("[^1]: The **footnote** with [[Page]]")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewFootnoteDefinition("1",
					content.NewText("The "),
					content.NewStrong(content.NewText("footnote")),
					content.NewText(" with "),
					content.NewPageLink("Page"),
				),
			)))
		})

		It("keeps a definition where it was written", func() {
			block, err := markdown.ParseString("Text[^1]\n\n[^1]: The footnote\n\nMore text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("Text"),
					content.NewFootnoteRef("1"),
				),
				content.NewFootnoteDefinition("1",
					content.NewText("The footnote"),
				),
				content.NewParagraph(
					content.NewText("More text"),
				),
			)))
		})

		It("can parse a footnote in a sub block", func() {
			block, err := markdown.ParseString("- Text[^1]\n  [^1]: The footnote")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewBlock(
					content.NewParagraph(
						content.NewText("Text"),
						content.NewFootnoteRef("1"),
					),
					content.NewFootnoteDefinition("1",
						content.NewText("The footnote"),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
				),
			)))
		})

		It("keeps a link with a caret as a link", func() {
			block, err := markdown.ParseString("[^1](https://example.com)")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewLink("https://example.com",
						content.NewText("^1"),
					),
				),
			)))
		})

		It("keeps brackets with a space in them as text", func() {
			block, err := markdown.ParseString("Text [^not a label] more")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("Text [^not a label] more"),
				),
			)))
		})
	})

	Describe("Math", func() {
		It("can parse inline math", func() {
			block, err := markdown.ParseString("The formula $E = mc^2$ is famous")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("The formula "),
					content.NewMath("E = mc^2"),
					content.NewText(" is famous"),
				),
			)))
		})

		It("can parse displayed math within a line", func() {
			block, err := markdown.ParseString("The formula $$E = mc^2$$ is famous")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("The formula "),
					content.NewDisplayedMath("E = mc^2"),
					content.NewText(" is famous"),
				),
			)))
		})

		It("can parse a math block", func() {
			block, err := markdown.ParseString("$$\nE = mc^2\n$$")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewMathBlock("E = mc^2\n"),
			)))
		})

		It("can parse a math block with several lines", func() {
			block, err := markdown.ParseString("$$\na = b\nc = d\n$$")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewMathBlock("a = b\nc = d\n"),
			)))
		})

		It("can parse a math block in a sub block", func() {
			block, err := markdown.ParseString("- Formula\n  $$\n  E = mc^2\n  $$")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewBlock(
					content.NewParagraph(
						content.NewText("Formula"),
					),
					content.NewMathBlock("E = mc^2\n").
						WithPreviousLineType(content.PreviousLineTypeNonBlank),
				),
			)))
		})

		It("keeps amounts in a sentence as text", func() {
			block, err := markdown.ParseString("It costs $5 and $10")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("It costs $5 and $10"),
				),
			)))
		})

		It("keeps a dollar sign followed by a space as text", func() {
			block, err := markdown.ParseString("Costs $ 5 or $ 10")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("Costs $ 5 or $ 10"),
				),
			)))
		})

		It("does not carry a formula past the line it starts on", func() {
			block, err := markdown.ParseString("Costs $5\nand $10 more")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("Costs $5").WithSoftLineBreak(),
					content.NewText("and $10 more"),
				),
			)))
		})

		It("keeps an escaped dollar sign as text", func() {
			block, err := markdown.ParseString("Not math: \\$x\\$")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("Not math: $x$"),
				),
			)))
		})
	})

	Describe("Tables", func() {
		It("can parse a table with a header and a row", func() {
			block, err := markdown.ParseString("| Name | Value |\n| --- | --- |\n| a | 1 |")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewTable(
					content.NewTableRow(
						content.NewTableCell(content.NewText("Name")),
						content.NewTableCell(content.NewText("Value")),
					),
					content.NewTableRow(
						content.NewTableCell(content.NewText("a")),
						content.NewTableCell(content.NewText("1")),
					),
				).WithAlignments(content.TableAlignmentNone, content.TableAlignmentNone),
			)))
		})

		It("can parse a table with only a header", func() {
			block, err := markdown.ParseString("| Name |\n| --- |")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewTable(
					content.NewTableRow(
						content.NewTableCell(content.NewText("Name")),
					),
				).WithAlignments(content.TableAlignmentNone),
			)))
		})

		It("can parse the alignment of the columns", func() {
			block, err := markdown.ParseString("| a | b | c | d |\n| :-- | --: | :-: | --- |")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewTable(
					content.NewTableRow(
						content.NewTableCell(content.NewText("a")),
						content.NewTableCell(content.NewText("b")),
						content.NewTableCell(content.NewText("c")),
						content.NewTableCell(content.NewText("d")),
					),
				).WithAlignments(
					content.TableAlignmentLeft,
					content.TableAlignmentRight,
					content.TableAlignmentCenter,
					content.TableAlignmentNone,
				),
			)))
		})

		It("can parse formatting in a cell", func() {
			block, err := markdown.ParseString("| Name |\n| --- |\n| **a** and [[Page]] |")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewTable(
					content.NewTableRow(
						content.NewTableCell(content.NewText("Name")),
					),
					content.NewTableRow(
						content.NewTableCell(
							content.NewStrong(content.NewText("a")),
							content.NewText(" and "),
							content.NewPageLink("Page"),
						),
					),
				).WithAlignments(content.TableAlignmentNone),
			)))
		})

		It("can parse an escaped pipe in a cell", func() {
			block, err := markdown.ParseString("| Name |\n| --- |\n| a \\| b |")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewTable(
					content.NewTableRow(
						content.NewTableCell(content.NewText("Name")),
					),
					content.NewTableRow(
						content.NewTableCell(content.NewText("a | b")),
					),
				).WithAlignments(content.TableAlignmentNone),
			)))
		})

		It("can parse a table after a paragraph", func() {
			block, err := markdown.ParseString("Text\n\n| Name |\n| --- |")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("Text"),
				),
				content.NewTable(
					content.NewTableRow(
						content.NewTableCell(content.NewText("Name")),
					),
				).WithAlignments(content.TableAlignmentNone),
			)))
		})

		It("keeps a lone dash as an empty block instead of a table", func() {
			block, err := markdown.ParseString("Test\n-\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("Test"),
				),
				content.NewBlock(),
			)))
		})
	})

	Describe("Thematic breaks", func() {
		It("can parse", func() {
			block, err := markdown.ParseString("Foo\n\n---\n\nBar\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("Foo"),
				),
				content.NewThematicBreak(),
				content.NewParagraph(
					content.NewText("Bar"),
				),
			)))
		})
	})

	Describe("Raw HTML", func() {
		It("can parse inline", func() {
			block, err := markdown.ParseString("This is a <b>bold</b> word.")
			Expect(err).ToNot(HaveOccurred())

			// Logseq reads a whole element as HTML, so it is one node.
			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is a "),
					content.NewRawHTML("<b>bold</b>"),
					content.NewText(" word."),
				),
			)))
		})

		It("can parse block", func() {
			block, err := markdown.ParseString("<p>This is a paragraph.</p>")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewRawHTMLBlock("<p>This is a paragraph.</p>"),
			)))
		})
	})

	Describe("Blocks", func() {
		It("can parse sub-blocks", func() {
			block, err := markdown.ParseString("- Item 1\n- Item 2\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewBlock(
					content.NewParagraph(
						content.NewText("Item 1"),
					),
				),
				content.NewBlock(
					content.NewParagraph(
						content.NewText("Item 2"),
					),
				),
			)))
		})

		It("can parse block with content and sub-blocks", func() {
			block, err := markdown.ParseString("This is a paragraph\n\n- Item 1\n- Item 2\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is a paragraph"),
				),
				content.NewBlock(
					content.NewParagraph(
						content.NewText("Item 1"),
					),
				).WithPreviousLineType(content.PreviousLineTypeBlank),
				content.NewBlock(
					content.NewParagraph(
						content.NewText("Item 2"),
					),
				),
			)))
		})

		It("can parse sub-blocks with sub-blocks", func() {
			block, err := markdown.ParseString("- Item 1\n  - Item 1.1\n  - Item 1.2\n- Item 2\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewBlock(
					content.NewParagraph(
						content.NewText("Item 1"),
					),
					content.NewBlock(
						content.NewParagraph(
							content.NewText("Item 1.1"),
						),
					),
					content.NewBlock(
						content.NewParagraph(
							content.NewText("Item 1.2"),
						),
					),
				),
				content.NewBlock(
					content.NewParagraph(
						content.NewText("Item 2"),
					),
				),
			)))
		})

		It("can parse empty sub-block", func() {
			block, err := markdown.ParseString("- Item 1\n-\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewBlock(
					content.NewParagraph(
						content.NewText("Item 1"),
					),
				),
				content.NewBlock(),
			)))
		})

		It("can parse empty sub-block with content in main block", func() {
			block, err := markdown.ParseString("Test\n-\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("Test"),
				),
				content.NewBlock(),
			)))
		})

		It("can parse empty sub-block with content in main block", func() {
			block, err := markdown.ParseString("Test  \n-\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("Test"),
				),
				content.NewBlock(),
			)))
		})

		It("non-breaking dash does not start block", func() {
			block, err := markdown.ParseString("[[Test]]-\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewPageLink("Test"),
					content.NewText("-"),
				),
			)))
		})

		Describe("Indented blocks", func() {
			It("bullets indented under the content of a page are children of the pre-block", func() {
				block, err := markdown.ParseString("Paragraph\n\n\t- Child 1\n\t- Child 2\n\t\t- Grandchild\n- Item\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewPreBlock(
						content.NewParagraph(
							content.NewText("Paragraph"),
						),
						content.NewBlock(
							content.NewParagraph(
								content.NewText("Child 1"),
							),
						).WithPreviousLineType(content.PreviousLineTypeBlank),
						content.NewBlock(
							content.NewParagraph(
								content.NewText("Child 2"),
							),
							content.NewBlock(
								content.NewParagraph(
									content.NewText("Grandchild"),
								),
							),
						),
					),
					content.NewBlock(
						content.NewParagraph(
							content.NewText("Item"),
						),
					),
				)))
				Expect(block.Blocks()[0].IsPreBlock()).To(BeTrue())
				Expect(block.Blocks()[1].IsPreBlock()).To(BeFalse())
			})

			It("bullets indented directly after a paragraph are children of the pre-block", func() {
				block, err := markdown.ParseString("Paragraph\n\t- Child\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewPreBlock(
						content.NewParagraph(
							content.NewText("Paragraph"),
						),
						content.NewBlock(
							content.NewParagraph(
								content.NewText("Child"),
							),
						),
					),
				)))
			})

			It("bullets indented under page properties are children of the pre-block", func() {
				block, err := markdown.ParseString("key:: value\nParagraph\n\n\t- Child\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewPreBlock(
						content.NewProperties(
							content.NewProperty("key", content.NewText("value")),
						),
						content.NewParagraph(
							content.NewText("Paragraph"),
						).WithPreviousLineType(content.PreviousLineTypeNonBlank),
						content.NewBlock(
							content.NewParagraph(
								content.NewText("Child"),
							),
						).WithPreviousLineType(content.PreviousLineTypeBlank),
					),
				)))
			})

			It("bullets indented by eight spaces under a block are its children", func() {
				block, err := markdown.ParseString("- Parent\n        - Child 1\n        - Child 2\n- Sibling\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewBlock(
						content.NewParagraph(
							content.NewText("Parent"),
						),
						content.NewBlock(
							content.NewParagraph(
								content.NewText("Child 1"),
							),
						),
						content.NewBlock(
							content.NewParagraph(
								content.NewText("Child 2"),
							),
						),
					),
					content.NewBlock(
						content.NewParagraph(
							content.NewText("Sibling"),
						),
					),
				)))
			})

			It("bullets indented by eight spaces under a child block are its children", func() {
				block, err := markdown.ParseString("- Parent\n  - Child\n          - Grandchild\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewBlock(
						content.NewParagraph(
							content.NewText("Parent"),
						),
						content.NewBlock(
							content.NewParagraph(
								content.NewText("Child"),
							),
							content.NewBlock(
								content.NewParagraph(
									content.NewText("Grandchild"),
								),
							),
						),
					),
				)))
			})

			It("bullets indented by eight spaces after a blank line under a block are its children", func() {
				block, err := markdown.ParseString("- Parent\n\n        - Child\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewBlock(
						content.NewParagraph(
							content.NewText("Parent"),
						),
						content.NewBlock(
							content.NewParagraph(
								content.NewText("Child"),
							),
						).WithPreviousLineType(content.PreviousLineTypeBlank),
					),
				)))
			})

			It("indented text that is not a bullet is a paragraph that keeps its indentation", func() {
				block, err := markdown.ParseString("Paragraph\n\n\ttext\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("Paragraph"),
					),
					content.NewParagraph(
						content.NewText("text"),
					).WithIndentation("\t"),
				)))
			})
		})

		Describe("Malformed blocks", func() {
			It("trailing content is added to last block", func() {
				block, err := markdown.ParseString("This is a paragraph\n\n- Item 1\n- Item 2\n\nThis is a trailing paragraph\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("This is a paragraph"),
					),
					content.NewBlock(
						content.NewParagraph(
							content.NewText("Item 1"),
						),
					).WithPreviousLineType(content.PreviousLineTypeBlank),
					content.NewBlock(
						content.NewParagraph(
							content.NewText("Item 2"),
						),
						content.NewParagraph(
							content.NewText("This is a trailing paragraph"),
						),
					),
				)))
			})
		})

		Describe("Properties", func() {
			It("can parse single properties", func() {
				block, err := markdown.ParseString("key:: value")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key", content.NewText("value")),
					),
				)))
			})

			It("can parse properties of blocks after page properties", func() {
				block, err := markdown.ParseString("title:: X\n\n- Block\n  id:: 123\n  collapsed:: true\n- Other\n  key:: value\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("title", content.NewText("X")).WithPageRefsIgnored(true),
					),
					content.NewBlock(
						content.NewParagraph(
							content.NewText("Block"),
						),
						content.NewProperties(
							content.NewProperty("id", content.NewText("123")).WithPageRefsIgnored(true),
							content.NewProperty("collapsed", content.NewText("true")).WithPageRefsIgnored(true),
						).WithPreviousLineType(content.PreviousLineTypeNonBlank),
					).WithPreviousLineType(content.PreviousLineTypeBlank),
					content.NewBlock(
						content.NewParagraph(
							content.NewText("Other"),
						),
						content.NewProperties(
							content.NewProperty("key", content.NewText("value")),
						).WithPreviousLineType(content.PreviousLineTypeNonBlank),
					),
				)))
			})

			It("can parse property with a dot in its name", func() {
				block, err := markdown.ParseString("item\nlogseq.order-list-type:: number")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("item"),
					),
					content.NewProperties(
						content.NewProperty("logseq.order-list-type", content.NewText("number")).WithPageRefsIgnored(true),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			DescribeTable("reads names that Logseq allows",
				func(name string) {
					block, err := markdown.ParseString(name + ":: value")
					Expect(err).ToNot(HaveOccurred())

					Expect(block).To(tests.EqualNode(content.NewBlock(
						content.NewProperties(
							content.NewProperty(name, content.NewText("value")),
						),
					)))
				},
				Entry("letters with accents", "ändring"),
				Entry("upper case letters", "Status"),
				Entry("letters that are not Latin", "日本語"),
				Entry("an emoji", "🔥"),
				Entry("only digits", "123"),
				Entry("an underscore", "my_prop"),
				Entry("a question mark", "done?"),
				Entry("an exclamation mark", "important!"),
				Entry("a plus", "c++"),
				Entry("an asterisk", "a*b"),
				Entry("emphasis markup", "a*b*c"),
				Entry("an underscore between words", "my_long_prop"),
				Entry("an ampersand", "a&b"),
				Entry("a less than sign", "a<b"),
				Entry("an equals sign", "a=b"),
				Entry("a dollar sign", "a$b"),
				Entry("a percent sign", "a%b"),
				Entry("an apostrophe", "it's"),
				Entry("a # after the first character", "a#b"),
				Entry("a namespace", "ns/name"),
				Entry("bold markers", "**b**"),
				Entry("a comma at the end", "b,"),
				Entry("commas that give the EDN reader an even number of forms", "b,c,d"),
				Entry("brackets that give the EDN reader an even number of forms", "b[c]d"),
				Entry("backslashes that give the EDN reader characters", `b\c\d`),
				Entry("an at sign as an EDN character", `c\@y`),
				Entry("a zero width space", "b\u200bc"),
			)

			DescribeTable("drops names that Logseq does not allow",
				func(name string) {
					input := name + ":: [[value]]"
					block, err := markdown.ParseString(input)
					Expect(err).ToNot(HaveOccurred())

					// Then the line is not a property, and its value refers
					// to no page
					if properties := block.FindProperties(); properties != nil {
						Expect(properties.Get(name)).To(BeEmpty())
					}
					Expect(block.Children().PageReferences()).To(BeEmpty())

					// And the text does not change on save
					Expect(parseAndOutput(input)).To(Equal(input))
				},
				Entry("a # as the first character", "#tag"),
				Entry("a comma", "a,b"),
				Entry("a no-break space", "b\u00a0c"),
				Entry("an ideographic space", "作者\u3000名"),
				Entry("brackets that give the EDN reader an odd number of forms", "b[[c]]"),
				Entry("a backslash at the end", `b\`),
				Entry("a semicolon", "a;b"),
				Entry("an at sign", "a@b"),
				Entry("a tilde", "a~b"),
				Entry("a backtick", "a`b"),
				Entry("a backslash", `a\b`),
				Entry("a quote", `a"b`),
				Entry("a vertical bar", "a|b"),
				Entry("a caret", "a^b"),
				Entry("parentheses", "a(b)"),
				Entry("braces", "a{b}"),
				Entry("a slash first", "/a"),
				Entry("a slash last", "a/"),
				Entry("two slashes", "a/b/c"),
				Entry("a digit after the slash", "a/1"),
			)

			It("keeps a line with a colon in the name as text", func() {
				// Given a name with a colon, which mldoc does not read as a name
				block, err := markdown.ParseString("a:b:: value")
				Expect(err).ToNot(HaveOccurred())

				// Then the line is not a property line
				Expect(block.FindProperties()).To(BeNil())
			})

			Describe("Property lines with a name that Logseq rejects", func() {
				// refTargets parses the input and returns the targets of all
				// page references in the first block.
				refTargets := func(input string) []string {
					root, err := markdown.ParseString(input)
					Expect(err).ToNot(HaveOccurred())
					Expect(root.Blocks()).ToNot(BeEmpty())

					targets := make([]string, 0)
					for _, ref := range root.Blocks()[0].Children().PageReferences() {
						targets = append(targets, ref.(content.PageRef).GetTo())
					}
					return targets
				}

				It("keeps the properties after a line with an invalid name in the same group", func() {
					// Given a property line with an invalid name between two valid ones
					input := "- a\n  b:: c\n  #d:: e\n  f:: g"

					root, err := markdown.ParseString(input)
					Expect(err).ToNot(HaveOccurred())

					// Then the block has the properties b and f, the same as Logseq
					properties := root.Blocks()[0].FindProperties()
					Expect(properties).ToNot(BeNil())
					Expect(properties.Get("b")).ToNot(BeEmpty())
					Expect(properties.Get("f")).ToNot(BeEmpty())

					// And the invalid line makes no tag
					Expect(refTargets(input)).ToNot(ContainElement("d"))

					// And the text does not change on save
					Expect(parseAndOutput(input)).To(Equal(input))
				})

				It("makes no reference from the value of a property line with an invalid name", func() {
					// Given a property line whose name has a parenthesis
					// Then Logseq drops the entry and its references
					Expect(refTargets("- a\n  b(x:: [[e]]")).To(BeEmpty())
				})

				It("still makes references from the valid properties around it", func() {
					// Given an invalid line between properties with links
					// Then only the valid properties ref their pages
					Expect(refTargets("- a\n  b:: [[c]]\n  a~b:: [[z]] #t\n  f:: [[g]]")).To(ConsistOf("c", "g"))
				})

				It("reads a line that starts with a page link as a line with an invalid name", func() {
					// Given a property line whose name is a page link
					input := "- a\n  [[b]]:: c\n  f:: g"

					// Then the link is not a reference, and f is a property
					Expect(refTargets(input)).To(BeEmpty())

					root, err := markdown.ParseString(input)
					Expect(err).ToNot(HaveOccurred())
					Expect(root.Blocks()[0].FindProperties().Get("f")).ToNot(BeEmpty())
					Expect(parseAndOutput(input)).To(Equal(input))
				})

				It("reads a group with only invalid names as the first group", func() {
					// Given an invalid property line, a line that is not a
					// property, and then a valid property
					input := "- x\n  ns/1x:: a\n  ns/1x::\"q\"\n  collapsed:: y"

					root, err := markdown.ParseString(input)
					Expect(err).ToNot(HaveOccurred())

					// Then collapsed is in the second group, which Logseq does
					// not read
					Expect(root.Blocks()[0].FindProperties().Get("collapsed")).To(BeEmpty())
					Expect(parseAndOutput(input)).To(Equal(input))
				})
			})

			It("can parse property with a value of several words", func() {
				block, err := markdown.ParseString("key:: a longer value")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key", content.NewText("a longer value")),
					),
				)))
			})

			It("can parse property with a value of several words followed by a paragraph", func() {
				block, err := markdown.ParseString("key:: a longer value\nThis is a paragraph\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key", content.NewText("a longer value")),
					),
					content.NewParagraph(
						content.NewText("This is a paragraph"),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("can parse multiple properties", func() {
				block, err := markdown.ParseString("key1:: value1\nkey2:: value2")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key1", content.NewText("value1")),
						content.NewProperty("key2", content.NewText("value2")),
					),
				)))
			})

			It("can parse property at the top of paragraph", func() {
				block, err := markdown.ParseString("key:: value\nThis is a paragraph\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key", content.NewText("value")),
					),
					content.NewParagraph(
						content.NewText("This is a paragraph"),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("can parse multiple properties at the top of paragraph", func() {
				block, err := markdown.ParseString("key1:: value1\nkey2:: value2\nThis is a paragraph\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key1", content.NewText("value1")),
						content.NewProperty("key2", content.NewText("value2")),
					),
					content.NewParagraph(
						content.NewText("This is a paragraph"),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("can parse property at the bottom of paragraph", func() {
				block, err := markdown.ParseString("This is a paragraph\nkey:: value\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("This is a paragraph"),
					),
					content.NewProperties(
						content.NewProperty("key", content.NewText("value")),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("can parse multiple properties at the bottom of paragraph", func() {
				block, err := markdown.ParseString("This is a paragraph\nkey1:: value1\nkey2:: value2\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("This is a paragraph"),
					),
					content.NewProperties(
						content.NewProperty("key1", content.NewText("value1")),
						content.NewProperty("key2", content.NewText("value2")),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("can parse property in the middle of paragraph", func() {
				block, err := markdown.ParseString("Line 1\nkey:: value\nLine 2")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("Line 1"),
					),
					content.NewProperties(
						content.NewProperty("key", content.NewText("value")),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
					content.NewParagraph(
						content.NewText("Line 2"),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("can parse multiple properties at the middle of paragraph", func() {
				block, err := markdown.ParseString("Line 1\nkey1:: value1\nkey2:: value2\nLine 2")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("Line 1"),
					),
					content.NewProperties(
						content.NewProperty("key1", content.NewText("value1")),
						content.NewProperty("key2", content.NewText("value2")),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
					content.NewParagraph(
						content.NewText("Line 2"),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("can parse property in top and end of paragraph", func() {
				block, err := markdown.ParseString("key1:: value1\nLine 1\nkey2:: value2")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key1", content.NewText("value1")),
					),
					content.NewParagraph(
						content.NewText("Line 1"),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
					content.NewProperties(
						content.NewProperty("key2", content.NewText("value2")),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank).WithPageRefsIgnored(true),
				)))
			})

			It("can parse multiple properties in top and end of paragraph", func() {
				block, err := markdown.ParseString("key1:: value1\nkey2:: value2\nLine 1\nkey3:: value3\nkey4:: value4")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key1", content.NewText("value1")),
						content.NewProperty("key2", content.NewText("value2")),
					),
					content.NewParagraph(
						content.NewText("Line 1"),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
					content.NewProperties(
						content.NewProperty("key3", content.NewText("value3")),
						content.NewProperty("key4", content.NewText("value4")),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank).WithPageRefsIgnored(true),
				)))
			})

			It("can parse property in top, middle and end of paragraph", func() {
				block, err := markdown.ParseString("key1:: value1\nLine 1\nkey2:: value2\nLine 2\nkey3:: value3")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key1", content.NewText("value1")),
					),
					content.NewParagraph(
						content.NewText("Line 1"),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
					content.NewProperties(
						content.NewProperty("key2", content.NewText("value2")),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank).WithPageRefsIgnored(true),
					content.NewParagraph(
						content.NewText("Line 2"),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
					content.NewProperties(
						content.NewProperty("key3", content.NewText("value3")),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank).WithPageRefsIgnored(true),
				)))
			})

			It("can parse property after paragraph", func() {
				block, err := markdown.ParseString("This is a paragraph\n\nkey:: value\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("This is a paragraph"),
					),
					content.NewProperties(
						content.NewProperty("key", content.NewText("value")),
					).WithPreviousLineType(content.PreviousLineTypeBlank),
				)))
			})

			It("will not parse property name no space after :: as property", func() {
				block, err := markdown.ParseString("key::value\nThis is a paragraph\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("key::value").WithSoftLineBreak(),
						content.NewText("This is a paragraph"),
					),
				)))
			})

			It("can parse property with an empty value", func() {
				block, err := markdown.ParseString("tags:: \nkey:: value\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("tags"),
						content.NewProperty("key", content.NewText("value")),
					),
				)))
			})

			It("can parse property with an empty value after a property that ends with a tag", func() {
				block, err := markdown.ParseString("type:: #Software\ntags:: \nkey:: value\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("type", content.NewHashtag("Software")),
						content.NewProperty("tags"),
						content.NewProperty("key", content.NewText("value")),
					),
				)))
			})

			It("can parse property with an empty value at the end of the input", func() {
				block, err := markdown.ParseString("key:: value\ntags::")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key", content.NewText("value")),
						content.NewProperty("tags").WithNoSpaceAfterSeparator(true),
					),
				)))
			})

			It("can parse property with non text nodes", func() {
				block, err := markdown.ParseString("key:: [[link]]\nThis is a paragraph\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key", content.NewPageLink("link")),
					),
					content.NewParagraph(
						content.NewText("This is a paragraph"),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("can parse property with mixed nodes", func() {
				block, err := markdown.ParseString("key:: [[link]] and #tag\nThis is a paragraph\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty(
							"key",
							content.NewPageLink("link"),
							content.NewText(" and "),
							content.NewHashtag("tag"),
						),
					),
					content.NewParagraph(
						content.NewText("This is a paragraph"),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("can parse property before blocks", func() {
				block, err := markdown.ParseString("key:: value\n- Item 1\n- Item 2\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key", content.NewText("value")),
					),
					content.NewBlock(
						content.NewParagraph(
							content.NewText("Item 1"),
						),
					),
					content.NewBlock(
						content.NewParagraph(
							content.NewText("Item 2"),
						),
					),
				)))
			})

			It("can parse property before unordered star list", func() {
				block, err := markdown.ParseString("key:: value\n* Item 1\n* Item 2\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key", content.NewText("value")),
					),
					content.NewList(
						content.ListTypeUnordered,
						content.NewListItem(
							content.NewParagraph(
								content.NewText("Item 1"),
							),
						),
						content.NewListItem(
							content.NewParagraph(
								content.NewText("Item 2"),
							),
						),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("keeps the value of a property that is not a list", func() {
				block, err := markdown.ParseString("key:: First, Second Name")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key", content.NewText("First, Second Name")),
					),
				)))
			})

			It("reads the values of a built-in list property as page references", func() {
				block, err := markdown.ParseString("alias:: First, Second Name")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty(
							"alias",
							content.NewPageRefText("First"),
							content.NewText(", "),
							content.NewPageRefText("Second Name"),
						),
					),
				)))
			})

			It("reads the single value of a list property as a page reference", func() {
				block, err := markdown.ParseString("tags:: Example")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("tags", content.NewPageRefText("Example")),
					),
				)))
			})

			It("reads the values of a list property split by a full width comma", func() {
				block, err := markdown.ParseString("tags:: First，Second")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty(
							"tags",
							content.NewPageRefText("First"),
							content.NewText("，"),
							content.NewPageRefText("Second"),
						),
					),
				)))
			})

			It("mixes page links and text in the values of a list property", func() {
				block, err := markdown.ParseString("tags:: [[First]], Second, #Third")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty(
							"tags",
							content.NewPageLink("First"),
							content.NewText(", "),
							content.NewPageRefText("Second"),
							content.NewText(", "),
							content.NewHashtag("Third"),
						),
					),
				)))
			})

			It("keeps a list property that has no values as it was written", func() {
				block, err := markdown.ParseString("tags:: ,")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("tags", content.NewText(",")),
					),
				)))
			})

			It("reads the values of a configured list property as page references", func() {
				block, err := markdown.ParseString(
					"key:: First, Second Name",
					markdown.WithPropertiesSeparatedByCommas("key"),
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty(
							"key",
							content.NewPageRefText("First"),
							content.NewText(", "),
							content.NewPageRefText("Second Name"),
						),
					),
				)))
			})

			It("matches the name of a list property without regard to case", func() {
				block, err := markdown.ParseString(
					"Key:: First",
					markdown.WithPropertiesSeparatedByCommas("KEY"),
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("Key", content.NewPageRefText("First")),
					),
				)))
			})

			It("marks a property whose references are ignored", func() {
				block, err := markdown.ParseString(
					"author:: Someone",
					markdown.WithIgnoredPageReferences("author"),
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("author", content.NewText("Someone")).
							WithPageRefsIgnored(true),
					),
				)))
			})

			It("keeps the page links of a property whose references are ignored", func() {
				block, err := markdown.ParseString(
					"author:: [[Someone]]",
					markdown.WithIgnoredPageReferences("author"),
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("author", content.NewPageLink("Someone")).
							WithPageRefsIgnored(true),
					),
				)))
			})

			It("does not split a list property whose references are ignored", func() {
				block, err := markdown.ParseString(
					"tags:: First, Second",
					markdown.WithIgnoredPageReferences("tags"),
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("tags", content.NewText("First, Second")).
							WithPageRefsIgnored(true),
					),
				)))
			})

			It("can parse property in block", func() {
				block, err := markdown.ParseString("- key:: value\nItem 1\n- Item 2\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewBlock(
						content.NewProperties(
							content.NewProperty("key", content.NewText("value")),
						),
						content.NewParagraph(
							content.NewText("Item 1"),
						).WithPreviousLineType(content.PreviousLineTypeNonBlank),
					),
					content.NewBlock(
						content.NewParagraph(
							content.NewText("Item 2"),
						),
					),
				)))
			})
		})
	})

	Describe("Advanced commands", func() {
		It("can parse", func() {
			block, err := markdown.ParseString("#+BEGIN_ABCDEF\nraw text\n#+END_ABCDEF\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewAdvancedCommand("ABCDEF", "raw text\n",
					content.NewParagraph(content.NewText("raw text")),
				),
			)))
		})

		It("can parse and keep indentation", func() {
			block, err := markdown.ParseString("#+BEGIN_ABCDEF\n  raw text\n#+END_ABCDEF\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewAdvancedCommand("ABCDEF", "  raw text\n",
					content.NewParagraph(content.NewText("raw text")),
				),
			)))
		})

		It("can parse multiple lines", func() {
			block, err := markdown.ParseString("#+BEGIN_ABCDEF\nraw text\nmore raw text\n#+END_ABCDEF\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewAdvancedCommand("ABCDEF", "raw text\nmore raw text\n",
					content.NewParagraph(
						content.NewText("raw text").WithSoftLineBreak(),
						content.NewText("more raw text"),
					),
				),
			)))
		})

		It("can parse multiple lines and keep indentation", func() {
			block, err := markdown.ParseString("#+BEGIN_ABCDEF\n  raw text\n  more raw text\n#+END_ABCDEF\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewAdvancedCommand("ABCDEF", "  raw text\n  more raw text\n",
					content.NewParagraph(
						content.NewText("raw text").WithSoftLineBreak(),
						content.NewText("more raw text"),
					),
				),
			)))
		})

		It("can parse multiple lines and keep indentation with empty lines", func() {
			block, err := markdown.ParseString("#+BEGIN_ABCDEF\n  raw text\n\n  more raw text\n#+END_ABCDEF\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewAdvancedCommand("ABCDEF", "  raw text\n\n  more raw text\n",
					content.NewParagraph(content.NewText("raw text")),
					content.NewParagraph(content.NewText("more raw text")),
				),
			)))
		})

		It("can parse multiple lines and keep indentation with empty lines at the end", func() {
			block, err := markdown.ParseString("#+BEGIN_ABCDEF\n  raw text\n  more raw text\n\n#+END_ABCDEF\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewAdvancedCommand("ABCDEF", "  raw text\n  more raw text\n\n",
					content.NewParagraph(
						content.NewText("raw text").WithSoftLineBreak(),
						content.NewText("more raw text"),
					),
				),
			)))
		})

		It("keeps indentation when parsed in a list", func() {
			block, err := markdown.ParseString("* Item\n  #+BEGIN_ABCDEF\n  raw text\n    raw text\n  #+END_ABCDEF\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewList(
					content.ListTypeUnordered,
					content.NewListItem(
						content.NewParagraph(
							content.NewText("Item"),
						),
						content.NewAdvancedCommand("ABCDEF", "raw text\n  raw text\n",
							content.NewParagraph(
								content.NewText("raw text").WithSoftLineBreak(),
								content.NewText("raw text"),
							),
						).
							WithPreviousLineType(content.PreviousLineTypeNonBlank),
					),
				),
			)))
		})

		It("trailing items in list parsed correctly", func() {
			block, err := markdown.ParseString("* Item\n  #+BEGIN_ABCDEF\n  raw text\n    raw text\n  #+END_ABCDEF\n* Item2")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewList(
					content.ListTypeUnordered,
					content.NewListItem(
						content.NewParagraph(
							content.NewText("Item"),
						),
						content.NewAdvancedCommand("ABCDEF", "raw text\n  raw text\n",
							content.NewParagraph(
								content.NewText("raw text").WithSoftLineBreak(),
								content.NewText("raw text"),
							),
						).
							WithPreviousLineType(content.PreviousLineTypeNonBlank),
					),
					content.NewListItem(
						content.NewParagraph(
							content.NewText("Item2"),
						),
					),
				),
			)))
		})

		Describe("Query", func() {
			It("can parse", func() {
				block, err := markdown.ParseString("#+BEGIN_QUERY\nraw text\n#+END_QUERY\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewQueryCommand("raw text\n"),
				)))
			})

			It("can parse and keep indentation", func() {
				block, err := markdown.ParseString("#+BEGIN_QUERY\n  raw text\n#+END_QUERY\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewQueryCommand("  raw text\n"),
				)))
			})

			It("can parse multiple lines", func() {
				block, err := markdown.ParseString("#+BEGIN_QUERY\nraw text\nraw text\n#+END_QUERY\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewQueryCommand("raw text\nraw text\n"),
				)))
			})

			It("can parse multiple lines with indentation", func() {
				block, err := markdown.ParseString("#+BEGIN_QUERY\nraw text\n  raw text\n#+END_QUERY\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewQueryCommand("raw text\n  raw text\n"),
				)))
			})

			It("keeps indentation when parsed in a list", func() {
				block, err := markdown.ParseString("* #+BEGIN_QUERY\n  raw text\n    raw text\n  #+END_QUERY\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewList(
						content.ListTypeUnordered,
						content.NewListItem(
							content.NewQueryCommand("raw text\n  raw text\n"),
						),
					),
				)))
			})
		})

		Describe("Quote", func() {
			It("can parse", func() {
				block, err := markdown.ParseString("#+BEGIN_QUOTE\nraw text\n#+END_QUOTE\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewAdvancedCommand("QUOTE", "raw text\n",
						content.NewParagraph(content.NewText("raw text")),
					),
				)))
			})

			It("can parse multiple lines", func() {
				block, err := markdown.ParseString("#+BEGIN_QUOTE\nraw text\nraw text\n#+END_QUOTE\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewAdvancedCommand("QUOTE", "raw text\nraw text\n",
						content.NewParagraph(
							content.NewText("raw text").WithSoftLineBreak(),
							content.NewText("raw text"),
						),
					),
				)))
			})
		})
	})

	Describe("Tasks", func() {
		Describe("Markers", func() {
			It("can parse TODO", func() {
				block, err := markdown.ParseString("TODO Task")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewText("Task"),
					),
				)))
			})

			It("can parse TODO with prefixed spaces", func() {
				block, err := markdown.ParseString("   TODO Task")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewText("Task"),
					),
				)))
			})

			It("skips TODO without space after", func() {
				block, err := markdown.ParseString("TODOTask")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("TODOTask"),
					),
				)))
			})

			It("skips TODO that is not first in paragraph", func() {
				block, err := markdown.ParseString("This is a TODO")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("This is a TODO"),
					),
				)))
			})

			It("skips TODO in second paragraph", func() {
				block, err := markdown.ParseString("Paragraph 1\n\nTODO Paragraph 2")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("Paragraph 1"),
					),
					content.NewParagraph(
						content.NewText("TODO Paragraph 2"),
					),
				)))
			})

			It("can parse DONE", func() {
				block, err := markdown.ParseString("DONE Task")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusDone),
						content.NewText("Task"),
					),
				)))
			})

			It("can parse DOING", func() {
				block, err := markdown.ParseString("DOING Task")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusDoing),
						content.NewText("Task"),
					),
				)))
			})

			It("can parse LATER", func() {
				block, err := markdown.ParseString("LATER Task")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusLater),
						content.NewText("Task"),
					),
				)))
			})

			It("can parse NOW", func() {
				block, err := markdown.ParseString("NOW Task")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusNow),
						content.NewText("Task"),
					),
				)))
			})

			It("can parse CANCELLED", func() {
				block, err := markdown.ParseString("CANCELLED Task")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusCancelled),
						content.NewText("Task"),
					),
				)))
			})

			It("can parse CANCELED", func() {
				block, err := markdown.ParseString("CANCELED Task")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusCanceled),
						content.NewText("Task"),
					),
				)))
			})

			It("can parse IN-PROGRESS", func() {
				block, err := markdown.ParseString("IN-PROGRESS Task")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusInProgress),
						content.NewText("Task"),
					),
				)))
			})

			It("can parse WAIT", func() {
				block, err := markdown.ParseString("WAIT Task")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusWait),
						content.NewText("Task"),
					),
				)))
			})

			It("can parse WAITING", func() {
				block, err := markdown.ParseString("WAITING Task")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusWaiting),
						content.NewText("Task"),
					),
				)))
			})
		})

		Describe("Priorities", func() {
			It("can parse a priority after a marker", func() {
				block, err := markdown.ParseString("TODO [#A] Task")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewTaskPriority(content.PriorityA),
						content.NewText("Task"),
					),
				)))
			})

			It("can parse a priority without a marker", func() {
				block, err := markdown.ParseString("[#B] Task")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskPriority(content.PriorityB),
						content.NewText("Task"),
					),
				)))
			})

			It("can parse a priority without any content after it", func() {
				block, err := markdown.ParseString("TODO [#C]")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewTaskPriority(content.PriorityC),
					),
				)))
			})

			It("keeps a priority that is not at the start as text", func() {
				block, err := markdown.ParseString("Task [#A] more")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("Task [#A] more"),
					),
				)))
			})

			It("keeps an unknown priority as text", func() {
				block, err := markdown.ParseString("TODO [#D] Task")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewText("[#D] Task"),
					),
				)))
			})

			It("keeps a tag in brackets as a tag", func() {
				block, err := markdown.ParseString("Task [#[[a tag]]] more")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("Task ["),
						content.NewHashtag("a tag"),
						content.NewText("] more"),
					),
				)))
			})
		})

		Describe("Scheduled and deadline", func() {
			It("can parse SCHEDULED", func() {
				block, err := markdown.ParseString("TODO Task\nSCHEDULED: <2024-01-15 Mon>")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewText("Task"),
					),
					content.NewScheduled(time.Date(2024, time.January, 15, 0, 0, 0, 0, time.Local)).
						WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("can parse DEADLINE", func() {
				block, err := markdown.ParseString("TODO Task\nDEADLINE: <2024-01-15 Mon>")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewText("Task"),
					),
					content.NewDeadline(time.Date(2024, time.January, 15, 0, 0, 0, 0, time.Local)).
						WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("can parse a date with a time of day", func() {
				block, err := markdown.ParseString("TODO Task\nSCHEDULED: <2024-01-15 Mon 09:30>")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewText("Task"),
					),
					content.NewScheduledWithTime(time.Date(2024, time.January, 15, 9, 30, 0, 0, time.Local)).
						WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("can parse a date with a repeater", func() {
				block, err := markdown.ParseString("TODO Task\nSCHEDULED: <2024-01-15 Mon .+3d>")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewText("Task"),
					),
					content.NewScheduled(time.Date(2024, time.January, 15, 0, 0, 0, 0, time.Local)).
						WithRepeater(content.NewRepeater(content.RepeaterTypeRestart, 3, content.RepeaterUnitDay)).
						WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("can parse a date with a time of day and a repeater", func() {
				block, err := markdown.ParseString("TODO Task\nDEADLINE: <2024-01-15 Mon 09:30 ++1w>")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewText("Task"),
					),
					content.NewDeadlineWithTime(time.Date(2024, time.January, 15, 9, 30, 0, 0, time.Local)).
						WithRepeater(content.NewRepeater(content.RepeaterTypeCatchUp, 1, content.RepeaterUnitWeek)).
						WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("can parse both a scheduled date and a deadline", func() {
				block, err := markdown.ParseString("TODO Task\nSCHEDULED: <2024-01-15 Mon>\nDEADLINE: <2024-01-20 Sat>")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewText("Task"),
					),
					content.NewScheduled(time.Date(2024, time.January, 15, 0, 0, 0, 0, time.Local)).
						WithPreviousLineType(content.PreviousLineTypeNonBlank),
					content.NewDeadline(time.Date(2024, time.January, 20, 0, 0, 0, 0, time.Local)).
						WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("keeps a blank line before the date when dates are relaxed", func() {
				// Logseq ignores a date after a blank line, so only relaxed
				// dates read it.
				block, err := markdown.ParseString("TODO Task\n\nSCHEDULED: <2024-01-15 Mon>", markdown.WithRelaxedTaskDates(true))
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewText("Task"),
					),
					content.NewScheduled(time.Date(2024, time.January, 15, 0, 0, 0, 0, time.Local)).
						WithPreviousLineType(content.PreviousLineTypeBlank),
				)))
			})

			It("reads the start of a time range, the same as Logseq", func() {
				block, err := markdown.ParseString("TODO Task\nSCHEDULED: <2024-01-15 Mon 10:00-11:00>")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewText("Task"),
					),
					content.NewScheduledWithTime(time.Date(2024, time.January, 15, 10, 0, 0, 0, time.Local)).
						WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)))
			})

			It("keeps a date that does not exist as text", func() {
				block, err := markdown.ParseString("TODO Task\nSCHEDULED: <2024-02-31 Sat>")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewText("Task").WithSoftLineBreak(),
						content.NewText("SCHEDULED: <2024-02-31 Sat>"),
					),
				)))
			})

			It("keeps text that only looks like a date as text", func() {
				block, err := markdown.ParseString("SCHEDULED: tomorrow")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("SCHEDULED: tomorrow"),
					),
				)))
			})
		})

		Describe("Logbooks", func() {
			logbookBlock := func(entries ...content.LogbookEntry) *content.Block {
				return content.NewBlock(
					content.NewParagraph(
						content.NewTaskMarker(content.TaskStatusTodo),
						content.NewText("Task"),
					),
					content.NewLogbook(entries...).
						WithPreviousLineType(content.PreviousLineTypeNonBlank),
				)
			}

			It("Can parse LOGBOOK", func() {
				block, err := markdown.ParseString("TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25:56]--[2023-06-26 Mon 17:25:56] =>  00:00:00\nCLOCK: [2023-06-26 Mon 17:25:57]--[2023-06-26 Mon 17:25:58] =>  00:00:01\n:END:")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(logbookBlock(
					content.NewLogbookEntryClock(
						time.Date(2023, time.June, 26, 17, 25, 56, 0, time.Local),
						time.Date(2023, time.June, 26, 17, 25, 56, 0, time.Local),
					),
					content.NewLogbookEntryClock(
						time.Date(2023, time.June, 26, 17, 25, 57, 0, time.Local),
						time.Date(2023, time.June, 26, 17, 25, 58, 0, time.Local),
					),
				)))
			})

			It("can parse a clock that is still running", func() {
				block, err := markdown.ParseString("TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25:56]\n:END:")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(logbookBlock(
					content.NewLogbookEntryClock(
						time.Date(2023, time.June, 26, 17, 25, 56, 0, time.Local),
						time.Time{},
					),
				)))
			})

			It("can parse a state change", func() {
				block, err := markdown.ParseString("TODO Task\n:LOGBOOK:\n* State \"DONE\" from \"TODO\" [2023-06-26 Mon 17:25:56]\n:END:")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(logbookBlock(
					content.NewLogbookEntryStateChange(
						content.TaskStatusTodo,
						content.TaskStatusDone,
						time.Date(2023, time.June, 26, 17, 25, 56, 0, time.Local),
					),
				)))
			})

			It("can parse a state change without a previous status", func() {
				block, err := markdown.ParseString("TODO Task\n:LOGBOOK:\n* State \"DONE\" [2023-06-26 Mon 17:25:56]\n:END:")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(logbookBlock(
					content.NewLogbookEntryStateChange(
						content.TaskStatusNone,
						content.TaskStatusDone,
						time.Date(2023, time.June, 26, 17, 25, 56, 0, time.Local),
					),
				)))
			})

			It("keeps an entry it does not understand as a raw entry", func() {
				block, err := markdown.ParseString("TODO Task\n:LOGBOOK:\nsomething else\n:END:")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(logbookBlock(
					content.NewLogbookEntryRaw("something else"),
				)))
			})

			It("can parse a clock written without seconds", func() {
				block, err := markdown.ParseString("TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25]--[2023-06-26 Mon 17:27] =>  00:02\n:END:")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(logbookBlock(
					content.NewLogbookEntryClock(
						time.Date(2023, time.June, 26, 17, 25, 0, 0, time.Local),
						time.Date(2023, time.June, 26, 17, 27, 0, 0, time.Local),
					),
				)))
			})

			It("can parse a running clock written without seconds", func() {
				block, err := markdown.ParseString("TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25]\n:END:")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(logbookBlock(
					content.NewLogbookEntryClock(
						time.Date(2023, time.June, 26, 17, 25, 0, 0, time.Local),
						time.Time{},
					),
				)))
			})

			It("can parse a state change written without seconds", func() {
				block, err := markdown.ParseString("TODO Task\n:LOGBOOK:\n* State \"DONE\" from \"TODO\" [2023-06-26 Mon 17:25]\n:END:")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(logbookBlock(
					content.NewLogbookEntryStateChange(
						content.TaskStatusTodo,
						content.TaskStatusDone,
						time.Date(2023, time.June, 26, 17, 25, 0, 0, time.Local),
					),
				)))
			})

			It("keeps a clock that ends before it starts as a raw entry", func() {
				block, err := markdown.ParseString("TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25:58]--[2023-06-26 Mon 17:25:56] =>  00:00:02\n:END:")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(logbookBlock(
					content.NewLogbookEntryRaw("CLOCK: [2023-06-26 Mon 17:25:58]--[2023-06-26 Mon 17:25:56] =>  00:00:02"),
				)))
			})

			It("keeps a state change with an unknown status as a raw entry", func() {
				block, err := markdown.ParseString("TODO Task\n:LOGBOOK:\n* State \"SOMEDAY\" from \"TODO\" [2023-06-26 Mon 17:25:56]\n:END:")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(logbookBlock(
					content.NewLogbookEntryRaw("* State \"SOMEDAY\" from \"TODO\" [2023-06-26 Mon 17:25:56]"),
				)))
			})
		})
	})

	Describe("Page links where Logseq does not read references", func() {
		// pageRefTitles returns the title of every page the parsed text refers to.
		pageRefTitles := func(input string) []string {
			block, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			titles := make([]string, 0)
			for _, ref := range block.Children().PageReferences() {
				titles = append(titles, ref.(content.PageRef).GetTo())
			}
			return titles
		}

		// Given a page link in a place that Logseq keeps as raw text or as a
		// label, When it is parsed, Then it refers to no page, the same as Logseq.
		DescribeTable("makes no page reference the same as Logseq",
			func(input string) {
				Expect(pageRefTitles(input)).To(BeEmpty())
			},
			Entry("in inline HTML", "- <span>[[a]]</span>"),
			Entry("in the label of a web link", "- [label [[a]]](http://x.com)"),
			Entry("in the label of a link to a path", "- [a [[b]] c](d)"),
			Entry("in a drawer", "- :mydrawer:\n  [[indrawer]]\n  :end:"),
			Entry("in the alt text of an image", "- ![alt [[inalt]]](a.png)"),
			Entry("in inline HTML with attributes", `- x <span class="y">[[a]]</span> z`),
			Entry("in inline HTML with tags in other case", "- <SPAN>[[a]]</span>"),
			Entry("in inline HTML in an element of the same tag", "- <span><span>[[a]]</span>[[b]]</span>"),
			Entry("in inline HTML that ends at a self closing tag", "- <span>[[a]] <br/>"),
			Entry("in inline HTML on two lines after the first line", "- a\n  x <span>\n  [[a]]</span>"),
			Entry("in a tag in inline HTML", "- <span>#a</span>"),
			Entry("in a tag in the label of a link", "- [a #b](http://x.com)"),
			Entry("in the label of a block reference", "- [x [[y]]](((64f0a2b1-1234-4abc-9def-0123456789ab)))"),
			Entry("in a drawer after text", "- a\n  text\n  :notes:\n  [[x]]\n  :END:"),
			Entry("in a drawer with blank lines and an end line with spaces", "- a\n  :notes:\n  [[x]]\n\n  [[y]]\n   :End: "),
		)

		DescribeTable("refers to the page, the same as Logseq",
			func(input string, expected []string) {
				Expect(pageRefTitles(input)).To(ConsistOf(expected))
			},
			Entry("after inline HTML", "- <b>x</b> [[a]]", []string{"a"}),
			Entry("after a void tag", "- <br> [[a]]", []string{"a"}),
			Entry("between two HTML elements", "- <span>a</span>[[b]]<span>c</span>", []string{"b"}),
			Entry("after an opening tag that is not closed", "- <span>[[a]]", []string{"a"}),
			Entry("in a tag that is not an HTML tag", "- <foo>[[a]]</foo>", []string{"a"}),
			Entry("in an element on the first line that is closed on the next line", "- x <span>\n  [[a]]</span>", []string{"a"}),
			Entry("after an element with an inner element that is not closed", "- <span>[[a]] <span>[[b]]</span> [[c]]", []string{"a", "c"}),
			Entry("after a drawer", "- a\n  :notes:\n  [[x]]\n  :end:\n  [[after]]", []string{"after"}),
			Entry("in a drawer line with a space after it", "- a\n  :notes: \n  [[x]]\n  :end:", []string{"x"}),
			Entry("in a drawer that is not closed in its block", "- a\n  :notes:\n  [[x]]\n- b\n  :end:", []string{"x"}),
			Entry("in a drawer name with a space", "- a\n  :my notes:\n  [[x]]\n  :end:", []string{"x"}),
		)

		It("reads a drawer with its name, content and end line", func() {
			// Given a block with a drawer
			block, err := markdown.ParseString("- a\n  :notes:\n  [[x]]\n\n  more\n  :end:")
			Expect(err).ToNot(HaveOccurred())

			// When it is parsed
			drawer, ok := block.Children().FindDeep(content.IsOfType[*content.Drawer]()).(*content.Drawer)

			// Then the drawer keeps its lines as text
			Expect(ok).To(BeTrue())
			Expect(drawer.Name).To(Equal("notes"))
			Expect(drawer.Content).To(Equal("[[x]]\n\nmore\n"))
			Expect(drawer.EndLine).To(Equal(":end:"))
		})

		It("refers only to the target of a labelled page link with a page link in the label", func() {
			// Given a labelled page link whose label has another page link
			// When it is parsed
			// Then only the target is referenced, the same as Logseq
			Expect(pageRefTitles("- [a [[b]] c]([[d]])")).To(ConsistOf("d"))
		})
	})

	Describe("Markdown links to Markdown files", func() {
		// pageRefTitles returns the title of every page the parsed text refers to.
		pageRefTitles := func(input string) []string {
			block, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			titles := make([]string, 0)
			for _, ref := range block.Children().PageReferences() {
				titles = append(titles, ref.(content.PageRef).GetTo())
			}
			return titles
		}

		It("refers to the page named by the label of a relative link to a .md file", func() {
			// Given a Markdown link to a relative .md file
			// When it is parsed
			// Then it refers to the page named by its label, the same as Logseq
			Expect(pageRefTitles("- [lbl](../pages/foo.md)")).To(ConsistOf("lbl"))
		})

		It("makes no page reference from a web link to a .md file", func() {
			// Given a Markdown link to a .md file on the web
			// When it is parsed
			// Then it refers to no page, the same as Logseq
			Expect(pageRefTitles("- [lbl](https://x.com/a.md)")).To(BeEmpty())
		})

		DescribeTable("refers to the page named by the label, the same as Logseq",
			func(input string, expected string) {
				Expect(pageRefTitles(input)).To(ConsistOf(expected))
			},
			Entry("a link to a file in the same folder", "- [lbl](page.md)", "lbl"),
			Entry("a link with file: and no slashes", "- [lbl](file:../pages/foo.md)", "lbl"),
			Entry("a link in angle brackets with a space", "- [lbl](<a b.md>)", "lbl"),
			Entry("a link that ends in upper case .MD", "- [lbl](a.MD)", "lbl"),
			Entry("a link to a .markdown file", "- [lbl](a.markdown)", "lbl"),
			Entry("a link with a title", `- [lbl](a.md "title")`, "lbl"),
			Entry("a label with spaces", "- [My page](a.md)", "My page"),
			Entry("a label with an escaped character", `- [x\*y](a.md)`, "x*y"),
			Entry("a label with brackets", "- [a [b] c](a.md)", "a [b] c"),
			Entry("a label that starts with a code span", "- [`x` y](a.md)", "x"),
			Entry("a link inside bold text", "- **see [lbl](a.md)**", "lbl"),
		)

		DescribeTable("makes no page reference, the same as Logseq",
			func(input string) {
				Expect(pageRefTitles(input)).To(BeEmpty())
			},
			Entry("a link with a protocol", "- [lbl](file:///a/foo.md)"),
			Entry("a link with an empty protocol", "- [lbl](://a.md)"),
			Entry("a link to a part of a file", "- [lbl](a.md#part)"),
			Entry("a link to a file named only .MD", "- [lbl](.MD)"),
			Entry("a link to a file of another type", "- [lbl](a.org)"),
			Entry("a link with an empty label", "- [](a.md)"),
			Entry("a label that is only emphasis", "- [*x*](a.md)"),
			Entry("a link in a block property", "- a\n  k:: [lbl](a.md)"),
			Entry("a link in a page property", "tags:: [lbl](a.md)\n\n- a"),
		)

		It("gives the label text before a code span as the title", func() {
			// Given a link to a .md file with text and then a code span in the label
			// When it is parsed
			// Then the title is the text before the code span, also its space, the same as Logseq
			Expect(pageRefTitles("- [a `x`](a.md)")).To(ConsistOf("a "))
		})
	})

	Describe("Block references directly after a word", func() {
		// blockRefIDs returns the id of every block reference in the parsed text.
		blockRefIDs := func(input string) []string {
			block, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			ids := make([]string, 0)
			for _, node := range block.Children().FilterDeep(content.IsOfType[*content.BlockRef]()) {
				ids = append(ids, node.(*content.BlockRef).ID)
			}
			return ids
		}

		It("makes no block reference directly after a colon", func() {
			// Given `ref:((uuid))` with no space before the parentheses
			// When it is parsed
			// Then there is no block reference, the same as Logseq
			Expect(blockRefIDs("- ref:((64f0a2b1-1234-4abc-9def-0123456789ab))")).To(BeEmpty())
		})

		It("reads a block reference after a space", func() {
			// Given `ref: ((uuid))` with a space before the parentheses
			// When it is parsed
			// Then it is a block reference
			Expect(blockRefIDs("- ref: ((64f0a2b1-1234-4abc-9def-0123456789ab))")).To(ConsistOf("64f0a2b1-1234-4abc-9def-0123456789ab"))
		})

		DescribeTable("block references that Logseq reads",
			func(input string) {
				// Given `((uuid))` after a character or element that ends a
				// run of plain text in Logseq
				// When it is parsed
				// Then it is a block reference
				Expect(blockRefIDs(input)).To(ConsistOf("64f0a2b1-1234-4abc-9def-0123456789ab"))
			},
			Entry("at the start", "- ((64f0a2b1-1234-4abc-9def-0123456789ab))"),
			Entry("after an underscore", "- a_((64f0a2b1-1234-4abc-9def-0123456789ab))"),
			Entry("after a page link", "- [[x]]((64f0a2b1-1234-4abc-9def-0123456789ab))"),
			Entry("after an escaped character", "- \\)((64f0a2b1-1234-4abc-9def-0123456789ab))"),
			Entry("after a timestamp", "- <2024-01-01 Mon>((64f0a2b1-1234-4abc-9def-0123456789ab))"),
			Entry("after a timestamp with a time and a repeater", "- <2024-01-01 Mon 10:00 .+1w>((64f0a2b1-1234-4abc-9def-0123456789ab))"),
			Entry("with a label", "- see[lbl](((64f0a2b1-1234-4abc-9def-0123456789ab)))"),
		)

		DescribeTable("text that Logseq does not read as a block reference",
			func(input string) {
				// Given `((uuid))` that continues a run of plain text
				// When it is parsed
				// Then there is no block reference
				Expect(blockRefIDs(input)).To(BeEmpty())
			},
			Entry("after a letter", "- a((64f0a2b1-1234-4abc-9def-0123456789ab))"),
			Entry("after a letter that is not ASCII", "- é((64f0a2b1-1234-4abc-9def-0123456789ab))"),
			Entry("after a closing parenthesis", "- )((64f0a2b1-1234-4abc-9def-0123456789ab))"),
			Entry("in three parentheses", "- (((64f0a2b1-1234-4abc-9def-0123456789ab)))"),
			Entry("after an escaped backslash", "- \\\\)((64f0a2b1-1234-4abc-9def-0123456789ab))"),
			Entry("after a timestamp directly after a word", "- a<2024-01-01 Mon>((64f0a2b1-1234-4abc-9def-0123456789ab))"),
		)

		DescribeTable("saves the text unchanged",
			func(input string) {
				// Given a block with `((uuid))` directly after other text
				block, err := markdown.ParseString(input)
				Expect(err).ToNot(HaveOccurred())

				// When it is written
				written, err := markdown.AsString(block)

				// Then the text is unchanged
				Expect(err).ToNot(HaveOccurred())
				Expect(written).To(Equal(input))
			},
			Entry("text after a word", "- ref:((64f0a2b1-1234-4abc-9def-0123456789ab))"),
			Entry("text in three parentheses", "- (((64f0a2b1-1234-4abc-9def-0123456789ab)))"),
			Entry("a reference after an escaped character", "- \\)((64f0a2b1-1234-4abc-9def-0123456789ab))"),
			Entry("a reference after a timestamp", "- <2024-01-01 Mon>((64f0a2b1-1234-4abc-9def-0123456789ab))"),
			Entry("a reference after a page link", "- [[x]]((64f0a2b1-1234-4abc-9def-0123456789ab))"),
			Entry("a reference on the next line", "- {\n  ((64f0a2b1-1234-4abc-9def-0123456789ab))"),
		)

		It("keeps a block reference after text that is built in code", func() {
			// Given text that ends in `)` and a block reference after it
			block := content.NewBlock(content.NewParagraph(
				content.NewText("see)"),
				content.NewBlockRef("64f0a2b1-1234-4abc-9def-0123456789ab"),
			))

			// When it is written and read again
			written, err := markdown.AsString(block)
			Expect(err).ToNot(HaveOccurred())

			// Then the `)` is escaped and the block reference is kept
			Expect(written).To(Equal("see\\)((64f0a2b1-1234-4abc-9def-0123456789ab))"))
			Expect(blockRefIDs("- " + written)).To(ConsistOf("64f0a2b1-1234-4abc-9def-0123456789ab"))
		})
	})

	Describe("Page links after an exclamation mark", func() {
		It("refers to the page of a page link after an exclamation mark", func() {
			// Given a block with `![[My Page]]`
			block, err := markdown.ParseString("- ![[My Page]]")
			Expect(err).ToNot(HaveOccurred())

			// When its page references are read
			titles := make([]string, 0)
			for _, ref := range block.Children().PageReferences() {
				titles = append(titles, ref.(content.PageRef).GetTo())
			}

			// Then it refers to `My Page`, the same as Logseq (text `!` and a page link)
			Expect(titles).To(ConsistOf("My Page"))
		})

		DescribeTable("reads the exclamation mark as text and the page link as a page link",
			func(input string, page string) {
				// Given a block with an exclamation mark directly before a page link
				block, err := markdown.ParseString(input)
				Expect(err).ToNot(HaveOccurred())

				// When it is parsed
				// Then it has the page link and no image
				link, ok := block.Children().FindDeep(content.IsOfType[*content.PageLink]()).(*content.PageLink)
				Expect(ok).To(BeTrue())
				Expect(link.To).To(Equal(page))
				Expect(block.Children().FilterDeep(content.IsOfType[*content.Image]())).To(BeEmpty())

				// And the exclamation mark is text before the page link
				text, ok := link.PreviousSibling().(*content.Text)
				Expect(ok).To(BeTrue())
				Expect(text.Value).To(HaveSuffix("!"))

				// And when it is written, Then the text is unchanged
				Expect(markdown.AsString(block)).To(Equal(input))
			},
			Entry("inside text", "- see ![[My Page]] here", "My Page"),
			Entry("with a file name", "- ![[a.png]]", "a.png"),
			Entry("with two exclamation marks", "- !![[x]]", "x"),
			Entry("with text in parentheses after it", "- ![[x]](url)", "x"),
		)

		It("still reads an image with a label", func() {
			// Given an image with a label that does not start with `[`
			block, err := markdown.ParseString("- ![a](b.png)")
			Expect(err).ToNot(HaveOccurred())

			// When it is parsed
			// Then it is an image and not a page link
			Expect(block.Children().FilterDeep(content.IsOfType[*content.Image]())).To(HaveLen(1))
			Expect(block.Children().FilterDeep(content.IsOfType[*content.PageLink]())).To(BeEmpty())
		})

		It("keeps an exclamation mark and a page link that is not closed as text", func() {
			// Given an exclamation mark and a page link without a closing `]]`
			input := "- ![[a]"
			block, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			// When it is parsed
			// Then there is no page link
			Expect(block.Children().FilterDeep(content.IsOfType[*content.PageLink]())).To(BeEmpty())

			// And when it is written, Then the text is unchanged
			Expect(markdown.AsString(block)).To(Equal(input))
		})
	})

	Describe("Highlights with equals signs", func() {
		It("reads text between double equals signs as a highlight", func() {
			// Given `==mark==`
			block, err := markdown.ParseString("==mark==")
			Expect(err).ToNot(HaveOccurred())

			// When it is parsed
			// Then it is a highlight with the text mark, the same as Logseq
			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewHighlight(content.NewText("mark")).WithEqualSigns(true),
				),
			)))
		})

		It("reads which marker a highlight is written with", func() {
			// Given a highlight with each marker
			block, err := markdown.ParseString("==mark== and ^^hi^^")
			Expect(err).ToNot(HaveOccurred())

			// When it is parsed
			highlights := block.Children().FilterDeep(content.IsOfType[*content.Highlight]())

			// Then only the first is marked as written with equals signs
			Expect(highlights).To(HaveLen(2))
			Expect(highlights[0].(*content.Highlight).EqualSigns).To(BeTrue())
			Expect(highlights[1].(*content.Highlight).EqualSigns).To(BeFalse())
		})

		DescribeTable("reads equals signs that are not a highlight as text",
			func(input string) {
				// Given equals signs that cannot open and close a highlight
				block, err := markdown.ParseString(input)
				Expect(err).ToNot(HaveOccurred())

				// When it is parsed
				// Then there is no highlight
				Expect(block.Children().FilterDeep(content.IsOfType[*content.Highlight]())).To(BeEmpty())
			},
			Entry("a comparison", "a == b"),
			Entry("a comparison in code", "if (a==b) then"),
			Entry("a line of equals signs", "Title\n==="),
			Entry("an opening pair only", "==x"),
		)
	})

	Describe("Image size metadata", func() {
		It("keeps the size after an image with the image and not as text", func() {
			// Given an image with the size Logseq writes when it is resized
			block, err := markdown.ParseString("![img](../assets/a.png){:height 100, :width 200}")
			Expect(err).ToNot(HaveOccurred())

			// When it is parsed
			// Then the paragraph holds only the image, as Logseq reads the size as
			// metadata of the image
			paragraph, ok := block.Children().FindDeep(content.IsOfType[*content.Paragraph]()).(*content.Paragraph)
			Expect(ok).To(BeTrue())
			Expect(paragraph.Children()).To(HaveLen(1))
			Expect(paragraph.Children()[0]).To(BeAssignableToTypeOf(&content.Image{}))
			Expect(paragraph.Children()[0].(*content.Image).Metadata).To(Equal("{:height 100, :width 200}"))
		})

		It("keeps the size of an image inside a link and inside bold text", func() {
			// Given images with a size inside a link and inside bold text
			block, err := markdown.ParseString("[![a](a.png){:width 1}](https://x.com) **![b](b.png){:width 2}**")
			Expect(err).ToNot(HaveOccurred())

			// When it is parsed
			// Then each image has its size
			images := block.Children().FilterDeep(content.IsOfType[*content.Image]())
			Expect(images).To(HaveLen(2))
			Expect(images[0].(*content.Image).Metadata).To(Equal("{:width 1}"))
			Expect(images[1].(*content.Image).Metadata).To(Equal("{:width 2}"))
		})

		DescribeTable("reads curly braces as text when they are not the size of an image",
			func(input string, text string) {
				// Given curly braces that are not directly after an image, or that do
				// not close on the line
				block, err := markdown.ParseString(input)
				Expect(err).ToNot(HaveOccurred())

				// When it is parsed
				// Then the image has no metadata, and the braces are text
				image, ok := block.Children().FindDeep(content.IsOfType[*content.Image]()).(*content.Image)
				Expect(ok).To(BeTrue())
				Expect(image.Metadata).To(BeEmpty())

				next, ok := image.NextSibling().(*content.Text)
				Expect(ok).To(BeTrue())
				Expect(next.Value).To(Equal(text))
			},
			Entry("after a space", "![a](a.png) {:width 1}", " {:width 1}"),
			Entry("without a closing brace", "![a](a.png){:width 1", "{:width 1"),
			Entry("with an escaped brace", `![a](a.png)\{:width 1}`, "{:width 1}"),
		)

		It("reads a macro directly after an image as a macro", func() {
			// Given a macro directly after an image
			block, err := markdown.ParseString("![a](a.png){{embed [[x]]}}")
			Expect(err).ToNot(HaveOccurred())

			// When it is parsed
			// Then the image has no metadata and the macro is kept
			image, ok := block.Children().FindDeep(content.IsOfType[*content.Image]()).(*content.Image)
			Expect(ok).To(BeTrue())
			Expect(image.Metadata).To(BeEmpty())
			Expect(image.NextSibling()).To(BeAssignableToTypeOf(&content.PageEmbed{}))
		})

		It("does not read curly braces after a link as metadata", func() {
			// Given curly braces directly after a link that is not an image
			block, err := markdown.ParseString("[a](https://x.com){:width 1}")
			Expect(err).ToNot(HaveOccurred())

			// When it is parsed
			// Then the braces are text
			link, ok := block.Children().FindDeep(content.IsOfType[*content.Link]()).(*content.Link)
			Expect(ok).To(BeTrue())
			Expect(link.NextSibling()).To(BeAssignableToTypeOf(&content.Text{}))
		})
	})
})

var _ = Describe("Spaces and alternative names for block ids", func() {
	const id = "6500e8a3-0000-4000-8000-000000000000"

	firstBlock := func(input string) *content.Block {
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(root.Blocks()).ToNot(BeEmpty())
		return root.Blocks()[0]
	}

	It("reads the id of a block written with two spaces after the colons", func() {
		// Given an id with two spaces after ::
		input := "- a\n  id::  " + id

		// Then the id is the uuid without a leading space, the same as Logseq
		Expect(firstBlock(input).ID()).To(Equal(id))

		// And the text does not change on save
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("reads custom_id as the id of the block", func() {
		// Given a block with custom_id, which Logseq reads as the id
		Expect(firstBlock("- a\n  custom_id:: " + id).ID()).To(Equal(id))
	})

	It("does not use an id that is not a uuid", func() {
		// Given an id that is not a uuid, which Logseq replaces
		// Then the block has no id from the text
		Expect(firstBlock("- a\n  id:: not-a-uuid").ID()).To(BeEmpty())
	})

	It("reads a value without the spaces that follow the colons", func() {
		// Given a value with two spaces after ::
		value := firstBlock("- a\n  b::  c").FindProperties().Get("b")

		// Then the value is c, the same as Logseq
		Expect(value).To(HaveLen(1))
		Expect(value[0]).To(tests.EqualNode(content.NewText("c")))
	})
})

var _ = Describe("Directives at the start of a page", func() {
	// pageRefTargets parses a page and returns the targets of all page
	// references in it.
	pageRefTargets := func(input string) []string {
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())

		targets := make([]string, 0)
		for _, ref := range root.Children().PageReferences() {
			targets = append(targets, ref.(content.PageRef).GetTo())
		}
		return targets
	}

	It("makes no tag from a title directive, the same as Logseq", func() {
		// Given a page that starts with #+title:
		// Then there is no reference to a page +title
		Expect(pageRefTargets("#+title: X\n\n- a")).ToNot(ContainElement("+title"))
	})

	It("makes no tags from directives and refs the tags they name", func() {
		// Given title and tags directives
		targets := pageRefTargets("#+title: X\n#+tags: a\n\n- a")

		// Then the directives make no tags, and the tag a is referenced
		Expect(targets).ToNot(ContainElement("+title"))
		Expect(targets).ToNot(ContainElement("+tags"))
		Expect(targets).To(ContainElement("a"))
	})

	It("makes no tag from a directive below the first block", func() {
		// Given a directive line at the start of a line after a block
		// Then Logseq reads a page property, not a tag
		Expect(pageRefTargets("- a\n#+foo: bar")).ToNot(ContainElement("+foo"))
	})

	It("does not change a directive on save", func() {
		input := "#+alias: Y\n- a"
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("keeps a directive in the text of a block as text, without a backslash on save", func() {
		// Given a directive line in the text of a block
		input := "- a\n  #+foo: bar\n  more"

		// Then it is not a tag, and it is written back as it was
		Expect(pageRefTargets(input)).To(BeEmpty())
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("reads a directive after a property line as a property of the same group", func() {
		// Given a block with a property line and then a directive line
		input := "- a\n  b:: c\n  #+foo: bar"
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())

		// Then the block has both properties, and the directive keeps its form
		properties := root.Blocks()[0].Properties()
		Expect(properties.GetAsNode("b")).ToNot(BeNil())
		foo := properties.GetAsNode("foo")
		Expect(foo).ToNot(BeNil())
		Expect(foo.Directive).To(BeTrue())
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	Describe("LaTeX math with backslash delimiters", func() {
		It("reads backslash parentheses as inline math", func() {
			// Given `\(z\)`
			block, err := markdown.ParseString("- \\(z\\)")
			Expect(err).ToNot(HaveOccurred())

			// When it is parsed
			// Then it is inline math with the formula z, the same as Logseq
			math, ok := block.Children().FindDeep(content.IsOfType[*content.Math]()).(*content.Math)
			Expect(ok).To(BeTrue(), "expected inline math")
			Expect(math.Value).To(Equal("z"))
		})

		It("makes no page references from links inside backslash parentheses", func() {
			// Given `\([[m2]]\)`, When it is parsed, Then the link is part of
			// the formula and refers to no page, the same as Logseq
			block, err := markdown.ParseString("- \\([[m2]]\\)")
			Expect(err).ToNot(HaveOccurred())

			Expect(block.Children().PageReferences()).To(BeEmpty())
		})

		It("makes no page references from the body of a LaTeX environment", func() {
			// Given a `\begin{equation}` block with a page link in its body
			block, err := markdown.ParseString("- latex env\n  \\begin{equation}\n  x = [[y]]\n  \\end{equation}")
			Expect(err).ToNot(HaveOccurred())

			// When it is parsed
			// Then the body is part of the formula and refers to no page
			Expect(block.Children().PageReferences()).To(BeEmpty())
		})
	})

	Describe("Begin and end blocks with parameters or in lower case", func() {
		// pageRefTitles returns the title of every page the parsed text refers to.
		pageRefTitles := func(input string) []string {
			block, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			titles := make([]string, 0)
			for _, ref := range block.Children().PageReferences() {
				titles = append(titles, ref.(content.PageRef).GetTo())
			}
			return titles
		}

		It("reads a source block with a language as one source block", func() {
			// Given a block with `#+BEGIN_SRC python`
			block, err := markdown.ParseString("- s\n  #+BEGIN_SRC python\n  x [[p]]\n  #+END_SRC")
			Expect(err).ToNot(HaveOccurred())

			// When it is parsed
			// Then it holds one SRC block construct, the same as Logseq
			commands := block.Children().FilterDeep(content.IsOfType[*content.AdvancedCommand]())
			Expect(commands).To(HaveLen(1))
			Expect(commands[0].(*content.AdvancedCommand).Type).To(MatchRegexp("(?i)^src$"))
		})

		It("makes no page references from a source block with a language", func() {
			// Given source blocks with a language, When they are parsed,
			// Then the code and the begin and end lines refer to no page
			Expect(pageRefTitles("- s\n  #+BEGIN_SRC python\n  x [[p]]\n  #+END_SRC")).To(BeEmpty())
			Expect(pageRefTitles("- #+BEGIN_SRC js\n  [[y]]\n  #+END_SRC")).To(BeEmpty())
		})

		It("makes no tags from the begin and end lines of a lower case quote block", func() {
			// Given a block with `#+begin_quote` and `#+end_quote`
			// When it is parsed
			// Then the begin and end lines are not tags, the same as Logseq
			titles := pageRefTitles("- n\n  #+begin_quote\n  q\n  #+end_quote")
			Expect(titles).ToNot(ContainElement("+begin_quote"))
			Expect(titles).ToNot(ContainElement("+end_quote"))
		})

		It("keeps the language of a source block as its parameters", func() {
			// Given a source block with a language and an option
			block, err := markdown.ParseString("- s\n  #+begin_src python -n\n  x\n  #+end_src")
			Expect(err).ToNot(HaveOccurred())

			// Then the type, the parameters and the case are kept
			command := block.Children().FindDeep(content.IsOfType[*content.AdvancedCommand]()).(*content.AdvancedCommand)
			Expect(command.Type).To(Equal("src"))
			Expect(command.Parameters).To(Equal("python -n"))
			Expect(command.LowerCase).To(BeTrue())
			Expect(command.Value).To(Equal("x\n"))
		})

		It("reads the text of a lower case quote block as Markdown", func() {
			// Given a lower case quote block with a page link
			// Then the block refers to the page, the same as Logseq
			Expect(pageRefTitles("- n\n  #+begin_quote\n  q [[p]]\n  #+end_quote")).To(ConsistOf("p"))
		})

		It("ends a block at an END line in another case", func() {
			// Given a note block that starts in mixed case and ends in upper case
			// Then it is one note block that refers to the page, the same as Logseq
			Expect(pageRefTitles("- n\n  #+Begin_Note\n  q [[p]]\n  #+END_NOTE")).To(ConsistOf("p"))
		})

		It("reads a lower case query block as a query", func() {
			// Given a lower case query block
			block, err := markdown.ParseString("- n\n  #+begin_query\n  {}\n  #+end_query")
			Expect(err).ToNot(HaveOccurred())

			// Then it is a query
			Expect(block.Children().FilterDeep(content.IsOfType[*content.QueryCommand]())).To(HaveLen(1))
		})
	})

	Describe("References inside quote and admonition blocks", func() {
		// pageRefTitles returns the title of every page the parsed text refers to.
		pageRefTitles := func(input string) []string {
			block, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			titles := make([]string, 0)
			for _, ref := range block.Children().PageReferences() {
				titles = append(titles, ref.(content.PageRef).GetTo())
			}
			return titles
		}

		It("finds page and block references inside a quote block", func() {
			// Given a quote block with a page link, a tag and a block reference
			input := "- q\n  #+BEGIN_QUOTE\n  a [[page]] #tag ((6571b1a6-1111-4111-8111-111111111111))\n  #+END_QUOTE"

			// When it is parsed
			block, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			// Then the block refers to both pages and to the block, the same as Logseq
			Expect(pageRefTitles(input)).To(ConsistOf("page", "tag"))

			ids := make([]string, 0)
			for _, node := range block.Children().FilterDeep(content.IsOfType[*content.BlockRef]()) {
				ids = append(ids, node.(*content.BlockRef).ID)
			}
			Expect(ids).To(ConsistOf("6571b1a6-1111-4111-8111-111111111111"))
		})

		// Given an admonition block with a page link, When it is parsed, Then
		// the block refers to the page, the same as Logseq.
		DescribeTable("finds page references inside an admonition block",
			func(kind string) {
				input := "- n\n  #+BEGIN_" + kind + "\n  hello [[p]]\n  #+END_" + kind
				Expect(pageRefTitles(input)).To(ConsistOf("p"))
			},
			Entry("in a NOTE block", "NOTE"),
			Entry("in a TIP block", "TIP"),
			Entry("in a WARNING block", "WARNING"),
			Entry("in an IMPORTANT block", "IMPORTANT"),
			Entry("in a CAUTION block", "CAUTION"),
			Entry("in a PINNED block", "PINNED"),
			Entry("in a CENTER block", "CENTER"),
			Entry("in a VERSE block", "VERSE"),
		)

		// Given a verbatim block with a page link, When it is parsed, Then it
		// refers to no page, the same as Logseq.
		DescribeTable("makes no page references inside a verbatim block",
			func(kind string) {
				input := "- n\n  #+BEGIN_" + kind + "\n  [[p]]\n  #+END_" + kind
				Expect(pageRefTitles(input)).To(BeEmpty())
			},
			Entry("in a COMMENT block", "COMMENT"),
			Entry("in a SRC block", "SRC"),
		)
	})

	Describe("Nested page links", func() {
		// pageRefTitles returns the title of every page the parsed text refers to.
		pageRefTitles := func(input string) []string {
			block, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			titles := make([]string, 0)
			for _, ref := range block.Children().PageReferences() {
				titles = append(titles, ref.(content.PageRef).GetTo())
			}
			return titles
		}

		It("refers to the whole nested title and to the inner page", func() {
			// Given a block with a page link that has a page link inside it
			// When it is parsed
			// Then it refers to the outer title and to the inner page, the same as Logseq
			titles := pageRefTitles("- [[a [[b]]]]")
			Expect(titles).To(ConsistOf("a [[b]]", "b"))
			Expect(titles).ToNot(ContainElement("a [[b"))
		})

		It("refers to the whole nested title of an extended hashtag", func() {
			// Given a block with an extended hashtag that has a page link inside it
			// When it is parsed
			// Then the tag points at the whole nested title
			titles := pageRefTitles("- #[[a [[b]]]]")
			Expect(titles).To(ContainElement("a [[b]]"))
			Expect(titles).ToNot(ContainElement("a [[b"))
		})

		It("refers to each page linked in a nested title", func() {
			Expect(pageRefTitles("- [[a [[b]] c [[d]]]]")).To(ConsistOf("a [[b]] c [[d]]", "b", "d"))
		})

		It("refers to the pages of page links nested at any depth", func() {
			Expect(pageRefTitles("- [[a [[b [[c]]]]]]")).To(ConsistOf("a [[b [[c]]]]", "b [[c]]", "c"))
		})

		It("ends a page link at the first closing brackets when the brackets do not balance", func() {
			Expect(pageRefTitles("- [[a [[b]]")).To(ConsistOf("a [[b"))
		})

		It("refers to the nested title of a page link with a label", func() {
			Expect(pageRefTitles("- [x]([[a [[b]]]])")).To(ConsistOf("a [[b]]", "b"))
		})

		It("reads an embed of a nested title as a page embed", func() {
			block, err := markdown.ParseString("{{embed [[a [[b]]]]}}")
			Expect(err).ToNot(HaveOccurred())

			embed := block.Children()[0].(*content.Paragraph).Children()[0]
			Expect(embed).To(BeAssignableToTypeOf(&content.PageEmbed{}))
			Expect(embed.(*content.PageEmbed).To).To(Equal("a [[b]]"))
		})
	})

	Describe("Hashtags at the end of emphasis", func() {
		// pageRefTitles returns the title of every page the parsed text refers to.
		pageRefTitles := func(input string) []string {
			block, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			titles := make([]string, 0)
			for _, ref := range block.Children().PageReferences() {
				titles = append(titles, ref.(content.PageRef).GetTo())
			}
			return titles
		}

		It("keeps a hash inside italic text as text of the emphasis", func() {
			// Given italic text that ends with a hashtag
			// When it is parsed
			// Then the emphasis holds the text and there is no tag, the same as Logseq
			block, err := markdown.ParseString("*see #x*")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewEmphasis(content.NewText("see #x")),
				),
			)))
		})

		It("makes no page reference from a hash at the end of italic text", func() {
			Expect(pageRefTitles("- *see #x*")).To(BeEmpty())
		})

		It("does not take the closing marker of bold text into the tag", func() {
			// Given bold text that ends with a hashtag
			// When it is parsed
			// Then no page with the closing marker in its name is referenced
			Expect(pageRefTitles("- **Status: #done**")).ToNot(ContainElement("done**"))
		})

		It("makes no page reference from a hash in the middle of bold text", func() {
			Expect(pageRefTitles("- **a #x b**")).To(BeEmpty())
		})

		It("makes no page reference from a hash inside strikethrough or highlighted text", func() {
			Expect(pageRefTitles("- ~~a #x~~")).To(BeEmpty())
			Expect(pageRefTitles("- ^^a #x^^")).To(BeEmpty())
		})

		It("keeps the page link of a bracketed hashtag inside italic text", func() {
			// Given italic text with a hashtag in the form #[[name]]
			// When it is parsed
			// Then the hash is text and the brackets are a page link, the same as Logseq
			block, err := markdown.ParseString("*see #[[a b]]*")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewEmphasis(
						content.NewText("see #"),
						content.NewPageLink("a b"),
					),
				),
			)))
		})

		It("keeps a hashtag after the end of italic text", func() {
			Expect(pageRefTitles("- *a #x* b #y")).To(Equal([]string{"y"}))
		})

		It("keeps a marker that cannot close formatting in the tag name", func() {
			Expect(pageRefTitles("- _a #snake_case b")).To(Equal([]string{"snake_case"}))
		})

		It("keeps a marker in the tag name when no formatting is open", func() {
			Expect(pageRefTitles("- a #x* b")).To(Equal([]string{"x*"}))
		})
	})

	Describe("Block properties after a quote, table, list or HTML", func() {
		const id = "65e5a0b2-1c2d-4e3f-9a8b-0c1d2e3f4a5b"

		// firstBlock parses the text and returns its first block.
		firstBlock := func(input string) *content.Block {
			root, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())
			Expect(root.Blocks()).ToNot(BeEmpty())
			return root.Blocks()[0]
		}

		// Given a block whose content ends in a container followed by an id
		// property, When it is parsed, Then the id belongs to the block, the
		// same as Logseq.
		DescribeTable("reads the id of the block",
			func(input string) {
				Expect(firstBlock(input).ID()).To(Equal(id))
			},
			Entry("after a quote", "- > quoted\n  id:: "+id),
			Entry("after a list with stars", "- * star\n  id:: "+id),
			Entry("after an ordered list", "- 1. one\n  id:: "+id),
			Entry("after HTML", "- <div>x</div>\n  id:: "+id),
		)

		It("keeps a property line inside HTML that ends with a closing tag", func() {
			// Given a pre block with a line that looks like a property
			// When it is parsed
			block := firstBlock("- <pre>\n  id:: " + id + "\n  </pre>")

			// Then the line is a part of the HTML and not a property
			Expect(block.ID()).To(BeEmpty())
		})

		It("reads properties after a table as properties and not as table rows", func() {
			// Given a block with a table followed by id and collapsed properties
			root, err := markdown.ParseString("- | a | b |\n  |---|---|\n  | 1 | 2 |\n  id:: " + id + "\n  collapsed:: true\n\t- child")
			Expect(err).ToNot(HaveOccurred())
			block := root.Blocks()[0]

			// When it is parsed
			// Then both are properties of the block
			Expect(block.ID()).To(Equal(id))
			properties := block.FindProperties()
			Expect(properties).ToNot(BeNil())
			Expect(properties.Get("collapsed")).ToNot(BeEmpty())

			// And the table has a header and one body row
			Expect(block.Content().FilterDeep(content.IsOfType[*content.TableRow]())).To(HaveLen(2))
		})

		It("does not write properties after a table as table rows", func() {
			// Given a block with a table followed by id and collapsed properties
			root, err := markdown.ParseString("- | a | b |\n  |---|---|\n  | 1 | 2 |\n  id:: " + id + "\n  collapsed:: true\n\t- child")
			Expect(err).ToNot(HaveOccurred())

			// When it is written
			// Then no property is written as a table row
			written, err := markdown.AsString(root)
			Expect(err).ToNot(HaveOccurred())
			Expect(written).ToNot(ContainSubstring("| id::"))
			Expect(written).ToNot(ContainSubstring("| collapsed::"))
		})

		It("keeps the id of a block after a quote when an id is asked for", func() {
			// Given a block with a quote followed by an id property
			root, err := markdown.ParseString("- > quoted\n  id:: " + id)
			Expect(err).ToNot(HaveOccurred())

			// When WithID is called
			block := root.Blocks()[0].WithID()

			// Then the id is unchanged and only one id line is written
			Expect(block.ID()).To(Equal(id))

			written, err := markdown.AsString(root)
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Count(written, "id::")).To(Equal(1))
		})
	})
})

var _ = Describe("References in quoted property values", func() {
	// refTargets parses the input and returns the targets of all page
	// references in the first block.
	refTargets := func(input string) []string {
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(root.Blocks()).ToNot(BeEmpty())

		targets := make([]string, 0)
		for _, ref := range root.Blocks()[0].Children().PageReferences() {
			targets = append(targets, ref.(content.PageRef).GetTo())
		}
		return targets
	}

	It("makes no references from a value that is wrapped in quotes", func() {
		// Given a quoted alias value with a comma
		input := "- a\n  alias:: \"x, y\""

		// Then the value is text, the same as Logseq, and is not split
		Expect(refTargets(input)).To(BeEmpty())
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("makes no reference from a link inside a quoted value", func() {
		// Given a link wrapped in quotes
		input := "- a\n  foo:: \"[[x]]\""

		// Then the block has no reference to x
		Expect(refTargets(input)).To(BeEmpty())
	})

	It("makes no tag from a quoted tags value", func() {
		// Given a quoted tags value
		input := "- a\n  tags:: \"x\""

		// Then the block has no reference
		Expect(refTargets(input)).To(BeEmpty())
	})

	It("still splits a value where only a part is quoted", func() {
		// Given tags where only the last part is quoted
		input := "- a\n  tags:: x, \"y, z\""

		// Then Logseq splits on every comma, quotes do not protect a part
		Expect(refTargets(input)).To(ConsistOf("x", "\"y", "z\""))
	})
})

var _ = Describe("References inside markup in property values", func() {
	// refTargets parses the input and returns the targets of all page
	// references in the first block.
	refTargets := func(input string) []string {
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(root.Blocks()).ToNot(BeEmpty())

		targets := make([]string, 0)
		for _, ref := range root.Blocks()[0].Children().PageReferences() {
			targets = append(targets, ref.(content.PageRef).GetTo())
		}
		return targets
	}

	It("makes no reference from a link inside bold text in a value, the same as Logseq", func() {
		// Given a property value with a link inside bold text
		input := "- a\n  foo:: **[[x]]**"

		// Then Logseq only takes links at the top level of the value
		Expect(refTargets(input)).To(BeEmpty())
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("makes a reference from a link at the top level of a value", func() {
		// Given a value with a top level link and bold text
		// Then the block refs x
		Expect(refTargets("- a\n  foo:: [[x]] **b**")).To(ConsistOf("x"))
	})

	It("still makes a reference from a link inside bold text in the block text", func() {
		// Given a block whose text has a link inside bold text
		// Then the block refs x, as Logseq searches markup in block text
		Expect(refTargets("- **[[x]]**\n  foo:: bar")).To(ConsistOf("x"))
	})
})

var _ = Describe("Property drawers", func() {
	It("reads the id of a block from a property drawer", func() {
		// Given a block with its id in a :PROPERTIES: drawer
		input := "- a\n  :PROPERTIES:\n  :id: 6500e8a3-0000-4000-8000-000000000000\n  :END:"

		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())

		// Then the block has that id, the same as Logseq
		Expect(root.Blocks()[0].ID()).To(Equal("6500e8a3-0000-4000-8000-000000000000"))

		// And the text does not change on save
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("reads custom_id in a property drawer as the id of the block", func() {
		// Given a drawer with custom_id, which Logseq maps to id
		root, err := markdown.ParseString("- a\n  :PROPERTIES:\n  :custom_id: 6500e8a3-0000-4000-8000-000000000000\n  :END:")
		Expect(err).ToNot(HaveOccurred())

		// Then the block has that id
		Expect(root.Blocks()[0].ID()).To(Equal("6500e8a3-0000-4000-8000-000000000000"))
	})
})

var _ = Describe("References in values of built-in properties", func() {
	// refTargets parses the input and returns the targets of all page
	// references in the first block.
	refTargets := func(input string) []string {
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(root.Blocks()).ToNot(BeEmpty())

		targets := make([]string, 0)
		for _, ref := range root.Blocks()[0].Children().PageReferences() {
			targets = append(targets, ref.(content.PageRef).GetTo())
		}
		return targets
	}

	It("makes no reference from the value of template", func() {
		// Given a template property with a link
		// Then Logseq reads the value as text
		Expect(refTargets("- a\n  template:: [[x]]")).To(BeEmpty())
	})

	It("makes no reference from the value of title", func() {
		Expect(refTargets("- a\n  title:: [[x]]")).To(BeEmpty())
	})

	It("makes no reference from the value of collapsed", func() {
		Expect(refTargets("- a\n  collapsed:: [[x]]")).To(BeEmpty())
	})

	It("makes no reference from the values of icon, id and filters", func() {
		Expect(refTargets("- a\n  icon:: [[x]]")).To(BeEmpty())
		Expect(refTargets("- a\n  id:: [[x]]")).To(BeEmpty())
		Expect(refTargets("- a\n  filters:: [[x]]")).To(BeEmpty())
	})

	It("makes a reference from the value of tags", func() {
		// Given tags, which Logseq links
		// Then the block refs x
		Expect(refTargets("- a\n  tags:: [[x]]")).To(ConsistOf("x"))
	})

	DescribeTable("makes no reference from the value of other built-in properties",
		func(input string) {
			// Given a built-in property with a link in its value
			// When its references are read
			// Then there are none, as Logseq reads the value as text
			Expect(refTargets(input)).To(BeEmpty())
		},
		Entry("heading", "- a\n  heading:: [[x]]"),
		Entry("background-color", "- a\n  background-color:: [[x]]"),
		Entry("an underscore in the name", "- a\n  background_color:: [[x]]"),
		Entry("upper case letters in the name", "- a\n  Template:: [[x]]"),
		Entry("a dot in the name", "- a\n  logseq.order-list-type:: [[x]]"),
		Entry("a tag", "- a\n  icon:: #x"),
		Entry("a task marker", "- a\n  later:: [[x]]"),
	)

	DescribeTable("makes a reference from the value of linkable and other properties",
		func(input string) {
			// Given a property that Logseq links, or that is not built in
			// When its references are read
			// Then the value refs x
			Expect(refTargets(input)).To(ConsistOf("x"))
		},
		Entry("alias", "- a\n  alias:: [[x]]"),
		Entry("aliases", "- a\n  aliases:: [[x]]"),
		Entry("a property that is not built in", "- a\n  related:: [[x]]"),
	)

	It("makes no reference from the title of a page", func() {
		// Given a page with a title property that has a link
		root, err := markdown.ParseString("title:: [[x]]\nalias:: [[y]]\n\n- a", markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())

		// When the references of the page are read
		targets := make([]string, 0)
		for _, ref := range root.Children().PageReferences() {
			targets = append(targets, ref.(content.PageRef).GetTo())
		}

		// Then only the alias is a reference
		Expect(targets).To(ConsistOf("y"))
	})

	It("still adds the properties that the graph ignores references of", func() {
		// Given a graph that ignores the references of a property
		root, err := markdown.ParseString("- a\n  related:: [[x]]\n  template:: [[y]]", markdown.WithIgnoredPageReferences("related"))
		Expect(err).ToNot(HaveOccurred())

		// When the references are read
		// Then neither property makes a reference
		Expect(root.Blocks()[0].Children().PageReferences()).To(BeEmpty())
	})
})

var _ = Describe("References in a second property group", func() {
	// refTargets parses the input and returns the targets of all page
	// references in the first block.
	refTargets := func(input string, opts ...markdown.ParseOption) []string {
		root, err := markdown.ParseString(input, opts...)
		Expect(err).ToNot(HaveOccurred())
		Expect(root.Blocks()).ToNot(BeEmpty())

		targets := make([]string, 0)
		for _, ref := range root.Blocks()[0].Children().PageReferences() {
			targets = append(targets, ref.(content.PageRef).GetTo())
		}
		return targets
	}

	It("makes no reference from properties after text in the block, the same as Logseq", func() {
		// Given a block with a second group of properties after a text line
		input := "- a\n  b:: c\n  text\n  d:: [[e]]"

		// Then Logseq only reads the first group, so e is not referenced
		Expect(refTargets(input)).To(BeEmpty())

		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(root.Blocks()[0].FindProperties().Get("d")).To(BeEmpty())
	})

	It("makes no reference from properties after a blank line in the block", func() {
		// Given a second group of properties after a blank line
		// Then Logseq does not reference e
		Expect(refTargets("- a\n  b:: c\n\n  d:: [[e]]")).To(BeEmpty())
	})

	It("still makes references from the first group of properties", func() {
		// Given a block with a link in its first group and its second group
		// Then only the first group refs its page
		Expect(refTargets("- a\n  b:: [[c]]\n  text\n  d:: [[e]]")).To(ConsistOf("c"))
	})

	It("still makes references from the text between the groups", func() {
		// Given a link in the text between two groups of properties
		// Then the text refs its page
		Expect(refTargets("- a\n  b:: c\n  see [[x]]\n  d:: [[e]]")).To(ConsistOf("x"))
	})

	It("makes no reference from a second group of page properties", func() {
		// Given a page with page properties, text and more properties
		root, err := markdown.ParseString("title:: t\ntext\nrelated:: [[e]]\n\n- a", markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())

		// When the references before the first block are read
		// Then the page in the second group is not a reference
		Expect(root.Children().FilterDeep(content.IsOfType[*content.Properties]())).To(HaveLen(2))
		for _, node := range root.Children() {
			if _, ok := node.(*content.Block); ok {
				break
			}
			Expect(content.NodeList{node}.PageReferences()).To(BeEmpty())
		}
	})

	It("makes a reference from the first group of a child block", func() {
		// Given a child block whose only group of properties has a link
		// Then the child block refs its page
		root, err := markdown.ParseString("- a\n  b:: c\n  text\n  d:: [[e]]\n\t- child\n\t  f:: [[g]]")
		Expect(err).ToNot(HaveOccurred())
		child := root.Blocks()[0].Blocks()[0]
		Expect(child.Children().PageReferences()).To(HaveLen(1))
	})

	It("makes references from later groups when they are turned on", func() {
		// Given a second group of properties after a text line
		// When the text is read with later property group refs on
		// Then the page in the second group is a reference
		Expect(refTargets("- a\n  b:: c\n  text\n  d:: [[e]]", markdown.WithLaterPropertyGroupRefs(true))).To(ConsistOf("e"))
	})

	It("keeps the second group out of the properties of the block when later groups make references", func() {
		// Given a second group of properties after a text line
		root, err := markdown.ParseString("- a\n  b:: c\n  text\n  d:: [[e]]", markdown.WithLaterPropertyGroupRefs(true))
		Expect(err).ToNot(HaveOccurred())

		// Then d is still not a property of the block, the same as Logseq
		Expect(root.Blocks()[0].FindProperties().Get("d")).To(BeEmpty())
	})
})
