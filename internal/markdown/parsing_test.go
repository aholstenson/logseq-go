package markdown_test

import (
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
					content.NewAutoLink("https://example.com"),
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
					content.NewMacro("macro", "arg1", "arg2"),
				),
			)))
		})

		It("can parse macro with name and quoted arguments", func() {
			block, err := markdown.ParseString("{{macro \"arg1\", \"arg2\"}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro", "arg1", "arg2"),
				),
			)))
		})

		It("can parse macro with name and one quoted argument containing a comma", func() {
			block, err := markdown.ParseString("{{macro \"arg1,\", arg2}}")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewMacro("macro", "arg1,", "arg2"),
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
					content.NewMacro("macro", "arg1\""),
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

		It("can parse indented code block", func() {
			block, err := markdown.ParseString("    This is an indented code block\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewCodeBlock("This is an indented code block\n"),
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
						content.NewText("This is a blockquote").WithSoftLineBreak(),
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

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is a "),
					content.NewRawHTML("<b>"),
					content.NewText("bold"),
					content.NewRawHTML("</b>"),
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
				),
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
					),
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

			It("can parse properties with spaces and punctuation", func() {
				block, err := markdown.ParseString("key:: value with spaces - dashes (and more)")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewProperties(
						content.NewProperty("key", content.NewText("value with spaces - dashes (and more)")),
					),
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
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
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
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
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
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
					content.NewParagraph(
						content.NewText("Line 2"),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
					content.NewProperties(
						content.NewProperty("key3", content.NewText("value3")),
					).WithPreviousLineType(content.PreviousLineTypeNonBlank),
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
				content.NewAdvancedCommand("ABCDEF", "raw text\n"),
			)))
		})

		It("can parse and keep indentation", func() {
			block, err := markdown.ParseString("#+BEGIN_ABCDEF\n  raw text\n#+END_ABCDEF\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewAdvancedCommand("ABCDEF", "  raw text\n"),
			)))
		})

		It("can parse multiple lines", func() {
			block, err := markdown.ParseString("#+BEGIN_ABCDEF\nraw text\nmore raw text\n#+END_ABCDEF\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewAdvancedCommand("ABCDEF", "raw text\nmore raw text\n"),
			)))
		})

		It("can parse multiple lines and keep indentation", func() {
			block, err := markdown.ParseString("#+BEGIN_ABCDEF\n  raw text\n  more raw text\n#+END_ABCDEF\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewAdvancedCommand("ABCDEF", "  raw text\n  more raw text\n"),
			)))
		})

		It("can parse multiple lines and keep indentation with empty lines", func() {
			block, err := markdown.ParseString("#+BEGIN_ABCDEF\n  raw text\n\n  more raw text\n#+END_ABCDEF\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewAdvancedCommand("ABCDEF", "  raw text\n\n  more raw text\n"),
			)))
		})

		It("can parse multiple lines and keep indentation with empty lines at the end", func() {
			block, err := markdown.ParseString("#+BEGIN_ABCDEF\n  raw text\n  more raw text\n\n#+END_ABCDEF\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewAdvancedCommand("ABCDEF", "  raw text\n  more raw text\n\n"),
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
						content.NewAdvancedCommand("ABCDEF", "raw text\n  raw text\n"),
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
						content.NewAdvancedCommand("ABCDEF", "raw text\n  raw text\n"),
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
					content.NewAdvancedCommand("QUOTE", "raw text\n"),
				)))
			})

			It("can parse multiple lines", func() {
				block, err := markdown.ParseString("#+BEGIN_QUOTE\nraw text\nraw text\n#+END_QUOTE\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(tests.EqualNode(content.NewBlock(
					content.NewAdvancedCommand("QUOTE", "raw text\nraw text\n"),
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

		It("Can parse LOGBOOK", func() {
			block, err := markdown.ParseString("TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25:56]--[2023-06-26 Mon 17:25:56] =>  00:00:00\nCLOCK: [2023-06-26 Mon 17:25:57]--[2023-06-26 Mon 17:25:58] =>  00:00:01\n:END:")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(tests.EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewTaskMarker(content.TaskStatusTodo),
					content.NewText("Task"),
				),
				content.NewLogbook(
					content.NewLogbookEntryRaw("CLOCK: [2023-06-26 Mon 17:25:56]--[2023-06-26 Mon 17:25:56] =>  00:00:00"),
					content.NewLogbookEntryRaw("CLOCK: [2023-06-26 Mon 17:25:57]--[2023-06-26 Mon 17:25:58] =>  00:00:01"),
				).WithPreviousLineType(content.PreviousLineTypeNonBlank),
			)))
		})
	})
})
