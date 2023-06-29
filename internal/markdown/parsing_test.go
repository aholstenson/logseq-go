package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Parsing", func() {
	Describe("Basic content", func() {
		It("can parse text", func() {
			block, err := markdown.ParseString("This is some basic text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is some basic text"),
				),
			)))
		})

		It("can parse text with escaped characters", func() {
			block, err := markdown.ParseString("This is some basic text with \\*escaped\\* characters")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("This is some basic text with *escaped* characters"),
				),
			)))
		})

		It("can parse heading", func() {
			block, err := markdown.ParseString("# Headline")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewHeading(1, content.NewText("Headline")),
			)))
		})

		It("can parse heading level 2", func() {
			block, err := markdown.ParseString("## Headline")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewHeading(2, content.NewText("Headline")),
			)))
		})

		It("can parse heading and text", func() {
			block, err := markdown.ParseString("# Headline\n\nThis is some basic text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewHeading(1, content.NewText("Headline")),
				content.NewParagraph(
					content.NewText("This is some basic text"),
				),
			)))
		})

		It("can parse soft line breaks", func() {
			block, err := markdown.ParseString("This is some basic text\nwith a soft line break")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
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

		It("can parse code", func() {
			block, err := markdown.ParseString("This is `code` text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewCodeSpan("code\\"),
				),
			)))
		})

		It("can parse code with double backticks", func() {
			block, err := markdown.ParseString("``code``")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewCodeSpan("code"),
				),
			)))
		})

		It("can parse code with double backticks and inline backtick", func() {
			block, err := markdown.ParseString("``co`de``")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewCodeSpan("co`de"),
				),
			)))
		})

		It("can parse code with triple backticks", func() {
			block, err := markdown.ParseString("```code```")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewLink(
						"https://example.com",
						content.NewText("This is a link"),
					).WithTitle("Title's"),
				),
			)))
		})

		It("can parse autolink", func() {
			block, err := markdown.ParseString("<https://example.com>")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewAutoLink("https://example.com"),
				),
			)))
		})

		It("can parse autolink without brackets", func() {
			block, err := markdown.ParseString("https://example.com")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewAutoLink("https://example.com"),
				),
			)))
		})

		It("can parse simple tags", func() {
			block, err := markdown.ParseString("#tag followed by some text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewHashtag("tag"),
					content.NewText(" followed by some text"),
				),
			)))
		})

		It("can parse tags with spaces", func() {
			block, err := markdown.ParseString("#[[tag with spaces]] and some other content")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewHashtag("tag with spaces"),
					content.NewText(" and some other content"),
				),
			)))
		})

		It("can parse tag when end of line", func() {
			block, err := markdown.ParseString("#tag")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewHashtag("tag"),
				),
			)))
		})

		It("will skip tag with spaces if end not found", func() {
			block, err := markdown.ParseString("#[[tag with spaces")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewText("#[[tag with spaces"),
				),
			)))
		})

		It("can parse wiki-style link", func() {
			block, err := markdown.ParseString("[[This is a link]] and this is some text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewPageLink("This is a link"),
					content.NewText(" and this is some text"),
				),
			)))
		})

		It("can parse wiki-style link with ] in it", func() {
			block, err := markdown.ParseString("[[This is ]a link]] and this is some text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewPageLink("This is ]a link"),
					content.NewText(" and this is some text"),
				),
			)))
		})

		It("can parse wiki-style link with escaped ]]", func() {
			block, err := markdown.ParseString("[[This is \\]]a link]] and this is some text")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewPageLink("This is ]]a link"),
					content.NewText(" and this is some text"),
				),
			)))
		})
	})

	Describe("Images", func() {
		It("can parse image", func() {
			block, err := markdown.ParseString("![This is an image](https://example.com/image.png)")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(
					content.NewImage(
						"https://example.com/image.png",
						content.NewText("This is an image"),
					).WithTitle("Title"),
				),
			)))
		})
	})

	Describe("Code blocks", func() {
		It("can parse code block", func() {
			block, err := markdown.ParseString("```\nThis is a code block\n```")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewCodeBlock("This is a code block\n"),
			)))
		})

		It("can parse code block with language", func() {
			block, err := markdown.ParseString("```go\nThis is a code block\n```")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewCodeBlock("This is a code block\n").WithLanguage("go"),
			)))
		})

		It("can parse indented code block", func() {
			block, err := markdown.ParseString("    This is an indented code block\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewCodeBlock("This is an indented code block\n"),
			)))
		})
	})

	Describe("Blockquotes", func() {
		It("can parse blockquote", func() {
			block, err := markdown.ParseString("> This is a blockquote\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewBlockquote(
					content.NewHeading(1,
						content.NewText("This is a header"),
					),
					content.NewParagraph(
						content.NewText("And this is a paragraph"),
					),
				),
			)))
		})
	})

	Describe("Lists", func() {
		Describe("Unordered lists", func() {
			Describe("With stars", func() {
				It("can parse", func() {
					block, err := markdown.ParseString("* Item 1\n* Item 2\n")
					Expect(err).ToNot(HaveOccurred())

					Expect(block).To(EqualNode(content.NewBlock(
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

					Expect(block).To(EqualNode(content.NewBlock(
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

					Expect(block).To(EqualNode(content.NewBlock(
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

					Expect(block).To(EqualNode(content.NewBlock(
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
			})
		})

		Describe("Ordered lists", func() {
			It("can parse", func() {
				block, err := markdown.ParseString("1. Item 1\n2. Item 2\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(EqualNode(content.NewBlock(
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

				Expect(block).To(EqualNode(content.NewBlock(
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

				Expect(block).To(EqualNode(content.NewBlock(
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

				Expect(block).To(EqualNode(content.NewBlock(
					content.NewList(
						content.ListTypeUnordered,
						content.NewListItem(
							content.NewParagraph(
								content.NewText("Item 1"),
							),
							content.NewList(
								content.ListTypeUnordered,
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
				block, err := markdown.ParseString("*\n\t* Item 1.1\n\t* Item 2\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(EqualNode(content.NewBlock(
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

				Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewRawHTMLBlock("<p>This is a paragraph.</p>"),
			)))
		})
	})

	Describe("Blocks", func() {
		It("can parse sub-blocks", func() {
			block, err := markdown.ParseString("- Item 1\n- Item 2\n")
			Expect(err).ToNot(HaveOccurred())

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
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

			Expect(block).To(EqualNode(content.NewBlock(
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

		Describe("Malformed blocks", func() {
			It("trailing content is added to last block", func() {
				block, err := markdown.ParseString("This is a paragraph\n\n- Item 1\n- Item 2\n\nThis is a trailing paragraph\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(EqualNode(content.NewBlock(
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
			It("can parse property at the top of first paragraph", func() {
				block, err := markdown.ParseString("key:: value\nThis is a paragraph\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewProperties(
							content.NewProperty("key", content.NewText("value")),
						),
						content.NewText("This is a paragraph"),
					),
				)))
			})

			It("can parse property at the bottom of first paragraph", func() {
				block, err := markdown.ParseString("This is a paragraph\nkey:: value\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("This is a paragraph").WithSoftLineBreak(),
						content.NewProperties(
							content.NewProperty("key", content.NewText("value")),
						),
					),
				)))
			})

			It("can parse property in the middle of first paragraph", func() {
				block, err := markdown.ParseString("Line 1\nkey:: value\nLine 2")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewText("Line 1").WithSoftLineBreak(),
						content.NewProperties(
							content.NewProperty("key", content.NewText("value")),
						),
						content.NewText("Line 2"),
					),
				)))
			})

			It("can parse property before blocks", func() {
				block, err := markdown.ParseString("key:: value\n- Item 1\n- Item 2\n")
				Expect(err).ToNot(HaveOccurred())

				Expect(block).To(EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewProperties(
							content.NewProperty("key", content.NewText("value")),
						),
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

				Expect(block).To(EqualNode(content.NewBlock(
					content.NewParagraph(
						content.NewProperties(
							content.NewProperty("key", content.NewText("value")),
						),
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
					),
				)))
			})
		})
	})
})

type equalNode struct {
	Expected content.Node
}

func EqualNode(expected content.Node) types.GomegaMatcher {
	return &equalNode{
		Expected: expected,
	}
}

func (matcher *equalNode) Match(actual interface{}) (bool, error) {
	if node, ok := actual.(content.Node); ok {
		return content.Debug(node) == content.Debug(matcher.Expected), nil
	}

	return false, nil
}

func (matcher *equalNode) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to equal", matcher.Expected)
}

func (matcher *equalNode) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to equal", matcher.Expected)
}

var _ types.GomegaMatcher = &equalNode{}