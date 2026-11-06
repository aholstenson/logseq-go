package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func parseAndOutput(input string, opts ...markdown.Option) string {
	block, err := markdown.ParseString(input)
	Expect(err).ToNot(HaveOccurred())
	v, err := markdown.AsString(block, opts...)
	Expect(err).ToNot(HaveOccurred())
	return v
}

func FullyEqual(name string, input string, opts ...markdown.Option) {
	It(name, func() {
		v := parseAndOutput(input, opts...)
		Expect(v).To(Equal(input))
	})
}

// FullyEqualWhenReadWith is FullyEqual for input that is read with the
// settings of a graph.
func FullyEqualWhenReadWith(name string, input string, parseOpts ...markdown.ParseOption) {
	It(name, func() {
		block, err := markdown.ParseString(input, parseOpts...)
		Expect(err).ToNot(HaveOccurred())

		v, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())
		Expect(v).To(Equal(input))
	})
}

func Varies(name string, input string, output string, opts ...markdown.Option) {
	It(name, func() {
		v := parseAndOutput(input, opts...)
		Expect(v).To(Equal(output))
	})
}

var _ = Describe("Parsing then outputting", func() {
	Describe("Paragraphs", func() {
		FullyEqual("Paragraph", "Basic content")
		FullyEqual("Paragraph with soft newline", "Basic\ncontent")
		FullyEqual("Paragraph with hard newline via backslash", "Basic\\\ncontent")
		Varies("Paragraph with hard newline via two spaces", "Basic  \ncontent", "Basic\\\ncontent")

		FullyEqual("Multiple paragraphs", "Basic content\n\nMore content")
	})

	Describe("Backslashes", func() {
		FullyEqual("Backslash before a letter", `- C:\Users\foo`)
		FullyEqual("Backslash at the end of a block with a child", "- C:\\Users\\\n\t- child")
		FullyEqual("Backslash at the end of a block before the next block", "- C:\\Users\\\n- next")
		FullyEqual("Escaped emphasis", `- \*not emphasis\*`)
		FullyEqual("Escaped tag", `- \#tag`)
		FullyEqual("Escaped code", "- \\`not code\\`")
		FullyEqual("Backslash in a property value", "- a\n  path:: C:\\Users\\x")
	})

	Describe("Inline formatting", func() {
		FullyEqual("Bold text", "**Basic** content")
		FullyEqual("Bold text with newline", "**Basic\ncontent**")
		FullyEqual("Bold text with hard newline", "**Basic\\\ncontent**")
		FullyEqual("Bold text with space before closing marker", "**Card from [[A]] and [[B]] **")
		FullyEqual("Italic text with space before closing marker", "*Basic *")
		FullyEqual("Strikethrough text with space before closing marker", "~~Basic ~~")
		FullyEqual("Highlighted text with space before closing marker", "^^Basic ^^")
		Varies("Bold text with hard newline via two spaces", "**Basic  \ncontent**", "**Basic\\\ncontent**")

		FullyEqual("Italic text", "*Basic* content")
		FullyEqual("Italic text with newline", "*Basic\ncontent*")
		FullyEqual("Italic text with hard newline", "*Basic\\\ncontent*")
		Varies("Italic text with hard newline via two spaces", "*Basic  \ncontent*", "*Basic\\\ncontent*")

		FullyEqual("Strikethrough text", "~~Basic~~ content")
		FullyEqual("Strikethrough text with newline", "~~Basic\ncontent~~")
		FullyEqual("Strikethrough text with hard newline", "~~Basic\\\ncontent~~")
		Varies("Strikethrough text with hard newline via two spaces", "~~Basic  \ncontent~~", "~~Basic\\\ncontent~~")
		FullyEqual("Strikethrough text containing escaped ~~", "~~Bas~\\~ic~~ content")

		FullyEqual("Highlighted text", "^^Basic^^ content")
		FullyEqual("Highlighted text with newline", "^^Basic\ncontent^^")
		FullyEqual("Highlighted text with hard newline", "^^Basic\\\ncontent^^")
		FullyEqual("Highlighted text containing escaped ^^", "^^Bas^\\^ic^^ content")
		FullyEqual("Text with a single ^", "2^10 content")

		// Code text maintains spaces and newlines
		FullyEqual("Code text", "`Basic` content")
		FullyEqual("Code text maintains newline", "`Basic\ncontent`")
		FullyEqual("Code text maintains spaces before 'hard' newline", "`Basic  \ncontent`")

		FullyEqual("Code text containing a backtick", "``Basic ` content``")
		FullyEqual("Code text starting with a backtick", "`` `Basic ``")
		FullyEqual("Code text ending with a backtick", "`` Basic` ``")
		FullyEqual("Code text of a single backtick", "`` ` ``")
		// Padding is only kept where it is needed to read the value back.
		Varies("Code text containing double backticks", "``` Basic ``content ```", "```Basic ``content```")
		FullyEqual("Code text surrounded by spaces", "`  Basic  `")
		FullyEqual("Code text of only spaces", "`  `")
	})

	Describe("Tags", func() {
		FullyEqual("Tag", "#tag and content")
		FullyEqual("Tag at end of line", "content #tag")
		FullyEqual("Tag with spaces", "#[[tag with spaces]] and content")
		FullyEqual("Page link with a label", "See [Goals]([[Workshop/Goals]]) here")
		FullyEqual("Page link with a label to a page with spaces", "See [the goals]([[Workshop Goals]]) here")
		FullyEqual("Page link with a label that has formatting", "[**Goals** of it]([[Workshop Goals]])")
		FullyEqual("Block reference with a label", "See [the block](((0b48a6c6-93ca-4d35-b945-6c59007f7962))) here")
		FullyEqual("Hash followed by a space", "issue # 5")
		FullyEqual("Hash at the end of a word", "C# is fine")
		FullyEqual("Hash at the end of a line", "content #")
		FullyEqual("Hash inside a word", "C#Sharp and C#, C")
		FullyEqual("Tag with trailing punctuation", "a #tag, b")
		FullyEqual("Hash in the text of a link", "**08:54** #Article [Smaller Teams:#](https://a.com/) #Readwise")
	})

	Describe("Heading", func() {
		FullyEqual("Heading 1", "# Heading")
		FullyEqual("Heading 2", "## Heading")
		FullyEqual("Heading 3", "### Heading")
		FullyEqual("Heading 4", "#### Heading")
		FullyEqual("Heading 5", "##### Heading")
		FullyEqual("Heading 6", "###### Heading")
		FullyEqual("Empty heading", "##")
		FullyEqual("Empty heading block", "- ##")
	})

	Describe("Heading combined with paragraph", func() {
		FullyEqual("Heading 1", "# Heading\n\nParagraph")
		FullyEqual("Heading 2", "## Heading\n\nParagraph")
		FullyEqual("Heading 3", "### Heading\n\nParagraph")
		FullyEqual("Heading 4", "#### Heading\n\nParagraph")
		FullyEqual("Heading 5", "##### Heading\n\nParagraph")
		FullyEqual("Heading 6", "###### Heading\n\nParagraph")
	})

	Describe("Code blocks", func() {
		FullyEqual("Code block", "```go\nfunc main() {\n\tfmt.Println(\"Hello world\")\n}\n```")
		FullyEqual("Code block with newline", "```go\nfunc main() {\n\tfmt.Println(\"Hello world\")\n}\n```\n\nParagraph")

		FullyEqual("Code block after paragraph", "Paragraph\n\n```go\nfunc main() {\n\tfmt.Println(\"Hello world\")\n}\n```")
		FullyEqual("Code block interrupting paragraph", "Paragraph\n```go\nfunc main() {\n\tfmt.Println(\"Hello world\")\n}\n```")
	})

	Describe("Footnotes", func() {
		FullyEqual("Footnote reference", "Task with a footnote[^1] in it")
		FullyEqual("Footnote reference with a named label", "Text[^note]")
		FullyEqual("Footnote definition", "[^1]: The footnote")
		FullyEqual("Footnote definition with formatting", "[^1]: The **footnote** with [[Page]]")
		FullyEqual("Footnote definition after a paragraph", "Text[^1]\n\n[^1]: The footnote")
		FullyEqual("Footnote definition in a block", "- Text[^1]\n  [^1]: The footnote")
		FullyEqual("Footnote definition over several lines", "[^1]: The\n    footnote")

		FullyEqual("Link with a caret stays a link", "[^1](https://example.com)")
		FullyEqual("Brackets with a space stay text", "Text [^not a label] more")
	})

	Describe("Math", func() {
		FullyEqual("Inline math", "The formula $E = mc^2$ is famous")
		FullyEqual("Displayed math within a line", "The formula $$E = mc^2$$ is famous")
		FullyEqual("Math block", "$$\nE = mc^2\n$$")
		FullyEqual("Math block with several lines", "$$\na = b\nc = d\n$$")
		FullyEqual("Math block after a paragraph", "Content\n\n$$\nE = mc^2\n$$")
		FullyEqual("Math block in a block", "- Formula\n  $$\n  E = mc^2\n  $$")

		FullyEqual("Amounts are not math", "It costs $5 and $10")
		FullyEqual("Dollar sign followed by a space is not math", "Costs $ 5 or $ 10")
		FullyEqual("Escaped math stays text", "Not math: \\$x\\$")
	})

	Describe("Tables", func() {
		FullyEqual("Table", "| Name | Value |\n| ---- | ----- |\n| a    | 1     |")
		FullyEqual("Table with only a header", "| Name |\n| ---- |")
		FullyEqual("Table with alignments", "| a   | b   | c   | d   |\n| :-- | --: | :-: | --- |")
		FullyEqual("Table with formatting in a cell", "| Name               |\n| ------------------ |\n| **a** and [[Page]] |")
		FullyEqual("Table with an escaped pipe in a cell", "| Name   |\n| ------ |\n| a \\| b |")
		FullyEqual("Table after a paragraph", "Content\n\n| a   |\n| --- |")
		FullyEqual("Table in a block", "- | a   |\n  | --- |")

		// The cells are padded to the width of the column they are in, so a
		// table that is written any other way lines up on the way out.
		Varies("Table without padding", "|Name|Value|\n|-|-|\n|a|1|", "| Name | Value |\n| ---- | ----- |\n| a    | 1     |")
		Varies("Table with extra padding", "| a      |\n| ------ |", "| a   |\n| --- |")

		// A row is read with the columns the header has, so cells that no
		// column belongs to are dropped and missing ones are filled in.
		Varies("Table with a row that is too long", "| a   |\n| --- |\n| b   | c |", "| a   |\n| --- |\n| b   |")
		Varies("Table with a row that is too short", "| a   | b   |\n| --- | --- |\n| c   |", "| a   | b   |\n| --- | --- |\n| c   |     |")
	})

	Describe("Macros", func() {
		FullyEqual("Macro with no arguments", "{{poem}}")
		FullyEqual("Macro with one argument", "{{poem red}}")
		FullyEqual("Macro with two arguments", "{{poem red, blue}}")
		Varies("Macro with two arguments, no space", "{{poem red,blue}}", "{{poem red, blue}}")
		FullyEqual("Macro with one argument and spaces", "{{poem red blue}}")
		FullyEqual("Macro with quoted argument with comma", "{{poem \"red, blue\"}}")

		Describe("Invalid", func() {
			FullyEqual("Macro without end", "{{poem red blue")
			FullyEqual("Macro with three curly braces missing the last closing brace", "{{{poem}}")
		})
	})

	Describe("Properties", func() {
		FullyEqual("Single property", "key:: value")
		FullyEqual("Property with a value of several words", "key:: a longer value")
		FullyEqual("Property with several values", "alias:: First, Second Name")
		FullyEqual("Property with several values as page links", "alias:: [[First]], [[Second Name]]")
		FullyEqual("Property with values that mix page links and text", "tags:: [[First]], Second, #Third")
		FullyEqual("Property with values separated by a full width comma", "tags:: First，Second")
		FullyEqual("Property with extra space around its values", "tags::  First ,  Second")
		FullyEqual("Property with values in brackets", "tags:: First, Second [bracketed]")
		FullyEqualWhenReadWith("Property with values of a configured list property",
			"key:: First, Second Name",
			markdown.WithPropertiesSeparatedByCommas("key"),
		)
		FullyEqualWhenReadWith("Property whose references are ignored",
			"author:: [[Someone]], Another",
			markdown.WithIgnoredPageReferences("author"),
		)
		FullyEqual("Multiple properties", "key:: value\nkey2:: value2")
		FullyEqual("Property with an empty value between other properties", "type:: #Software\ntags:: \nwebsite:: https://connectrpc.com/")
		Varies("Property with an empty value and no space after the name", "tags::\nkey:: value", "tags:: \nkey:: value")
		FullyEqual("Property with a dot in its name", "item\nlogseq.order-list-type:: number")
		FullyEqual("Properties with dots in their names", "logseq.color:: red\nlogseq.tldraw.page:: value")
		FullyEqual("Property with an underscore in its name", "my_prop:: value")
		FullyEqual("Property with letters that are not Latin in its name", "日本語:: value\nändring:: value")
		FullyEqual("Property with emphasis markup in its name", "a*b*c:: value")
		FullyEqual("Property with emphasis in its value", "key:: *value*")
		FullyEqual("Property with a namespace in its name", "ns/name:: value")
		FullyEqual("Property with a name that Logseq does not allow", "a@b:: value")

		It("writes a set property with a dot in its name", func() {
			block, err := markdown.ParseString("item")
			Expect(err).ToNot(HaveOccurred())

			block.Properties().Set("logseq.order-list-type", content.NewText("number"))

			v, err := markdown.AsString(block)
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal("item\nlogseq.order-list-type:: number"))
		})

		It("removes a property with a dot in its name and keeps the others", func() {
			block, err := markdown.ParseString("item\nkey:: value\nlogseq.order-list-type:: number\nother:: value2")
			Expect(err).ToNot(HaveOccurred())

			block.Properties().Remove("logseq.order-list-type")

			v, err := markdown.AsString(block)
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal("item\nkey:: value\nother:: value2"))
		})
		FullyEqual("Properties followed by trailing paragraph", "key:: value\nParagraph")
		FullyEqual("Paragraphs interrupted by properties", "Paragraph\nkey:: value")
		FullyEqual("Paragraphs interrupted by properties followed by more paragraph", "Paragraph\nkey:: value\nParagraph")
		FullyEqual("Paragraph followed by properties", "Paragraph\n\nkey:: value")
		FullyEqual("Properties followed by a blank line and a paragraph", "key:: value\n\nParagraph")
		FullyEqual("Properties followed by a blank line and blocks", "key:: value\n\n- Block 1\n- Block 2")
		FullyEqual("Properties followed by blocks", "key:: value\n- Block 1\n- Block 2")
	})

	Describe("Pre-block with children", func() {
		FullyEqual("Paragraph with indented children", "Paragraph\n\t- Child 1\n\t\t- Grandchild\n- Item")
		FullyEqual("Properties with indented children", "key:: value\nParagraph\n\t- Child\n- Item")
		FullyEqual("Properties with indented children followed by a blank line", "key:: value\n\t- Child\n\n- Item")
		Varies("Paragraph with indented children after a blank line",
			"Paragraph\n\n\t- Child\n- Item",
			"Paragraph\n\t- Child\n- Item",
		)
	})

	Describe("Tasks", func() {
		Describe("Markers", func() {
			FullyEqual("TODO Task", "TODO Task")
			FullyEqual("DOING Task", "DOING Task")
			FullyEqual("DONE Task", "DONE Task")
			FullyEqual("LATER Task", "LATER Task")
			FullyEqual("NOW Task", "NOW Task")
			FullyEqual("CANCELLED Task", "CANCELLED Task")
			FullyEqual("CANCELED Task", "CANCELED Task")
			FullyEqual("IN-PROGRESS Task", "IN-PROGRESS Task")
			FullyEqual("WAIT Task", "WAIT Task")
			FullyEqual("WAITING Task", "WAITING Task")

			Varies("Task with leading space", " TODO Task", "TODO Task")
		})

		Describe("Markers without text", func() {
			// A marker is written with a space after it, as Logseq reads a
			// marker that is followed by a line break as text.
			Varies("TODO on its own", "TODO", "TODO ")
			Varies("DONE on its own", "DONE", "DONE ")
			FullyEqual("TODO with trailing space", "TODO ")
			FullyEqual("TODO with trailing space and text on the next line", "TODO \nmore")

			// Without a space after it, the word is text in Logseq, and is
			// written as it is.
			FullyEqual("TODO with sub block", "- TODO\n\t- child")
			FullyEqual("TODO with text on the next line", "TODO\nmore")
			FullyEqual("TODO followed by formatting", "TODO**bold**")
		})

		Describe("Priorities", func() {
			FullyEqual("Priority A", "TODO [#A] Task")
			FullyEqual("Priority B", "TODO [#B] Task")
			FullyEqual("Priority C", "TODO [#C] Task")
			FullyEqual("Priority without a marker", "[#A] Task")
			FullyEqual("Priority without content after it", "TODO [#A]")
			FullyEqual("Priority in sub blocks", "- TODO [#A] Task\n- TODO [#B] Other")
			FullyEqual("Priority followed by a tag and a page link", "TODO [#A] #tag and [[page]]")
			FullyEqual("Priority with a scheduled date", "TODO [#A] Task\nSCHEDULED: <2024-01-15 Mon>")

			// Brackets that Logseq does not read as a priority stay as text.
			FullyEqual("Priority that is not at the start", "Task [#A] more")
			FullyEqual("Unknown priority", "TODO [#D] Task")
			FullyEqual("Priority not followed by a space", "Task [#A]more")
		})

		Describe("Logbooks", func() {
			FullyEqual("Clock", "TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25:56]--[2023-06-26 Mon 17:25:58] =>  00:00:02\n:END:")
			FullyEqual("Clock that is still running", "TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25:56]\n:END:")
			FullyEqual("Clock spanning more than a day", "TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 09:00:00]--[2023-06-27 Tue 11:30:45] =>  26:30:45\n:END:")
			FullyEqual("State change", "TODO Task\n:LOGBOOK:\n* State \"DONE\" from \"TODO\" [2023-06-26 Mon 17:25:56]\n:END:")
			FullyEqual("State change without a previous status", "TODO Task\n:LOGBOOK:\n* State \"DONE\" [2023-06-26 Mon 17:25:56]\n:END:")
			FullyEqual("State change followed by a clock", "TODO Task\n:LOGBOOK:\n* State \"DONE\" from \"TODO\" [2023-06-26 Mon 17:25:56]\nCLOCK: [2023-06-26 Mon 17:25:56]--[2023-06-26 Mon 17:25:58] =>  00:00:02\n:END:")

			// The day name is regenerated from the date, so a wrong one is
			// corrected on the way out.
			Varies(
				"Clock with the wrong day name",
				"TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Fri 17:25:56]\n:END:",
				"TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25:56]\n:END:",
			)

			// Entries that are not modelled are kept as they were found.
			FullyEqual("Unknown entry", "TODO Task\n:LOGBOOK:\nsomething else\n:END:")
			FullyEqual("State change with an unknown status", "TODO Task\n:LOGBOOK:\n* State \"SOMEDAY\" [2023-06-26 Mon 17:25:56]\n:END:")

			// Entries are written with the precision the graph is set up for,
			// so one written with the other precision is converted.
			Varies(
				"Clock without seconds",
				"TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25]--[2023-06-26 Mon 17:27] =>  00:02\n:END:",
				"TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25:00]--[2023-06-26 Mon 17:27:00] =>  00:02:00\n:END:",
			)
		})

		Describe("Logbooks without second support", func() {
			withoutSeconds := markdown.WithLogbookSeconds(false)

			FullyEqual("Clock", "TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25]--[2023-06-26 Mon 17:27] =>  00:02\n:END:", withoutSeconds)
			FullyEqual("Clock that is still running", "TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25]\n:END:", withoutSeconds)
			FullyEqual("Clock spanning more than a day", "TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 09:00]--[2023-06-27 Tue 11:30] =>  26:30\n:END:", withoutSeconds)
			FullyEqual("State change", "TODO Task\n:LOGBOOK:\n* State \"DONE\" from \"TODO\" [2023-06-26 Mon 17:25]\n:END:", withoutSeconds)

			Varies(
				"Clock with seconds",
				"TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25:56]--[2023-06-26 Mon 17:27:58] =>  00:02:02\n:END:",
				"TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25]--[2023-06-26 Mon 17:27] =>  00:02\n:END:",
				withoutSeconds,
			)
		})

		Describe("Scheduled and deadline", func() {
			FullyEqual("Scheduled", "TODO Task\nSCHEDULED: <2024-01-15 Mon>")
			FullyEqual("Deadline", "TODO Task\nDEADLINE: <2024-01-15 Mon>")
			FullyEqual("Scheduled with time", "TODO Task\nSCHEDULED: <2024-01-15 Mon 09:30>")
			FullyEqual("Scheduled with repeater", "TODO Task\nSCHEDULED: <2024-01-15 Mon .+3d>")
			FullyEqual("Scheduled with time and repeater", "TODO Task\nSCHEDULED: <2024-01-15 Mon 09:30 ++1w>")
			FullyEqual("Scheduled and deadline", "TODO Task\nSCHEDULED: <2024-01-15 Mon>\nDEADLINE: <2024-01-20 Sat>")
			FullyEqual("Scheduled followed by logbook", "TODO Task\nSCHEDULED: <2024-01-15 Mon>\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25:56]\n:END:")
			FullyEqual("Scheduled in sub blocks", "- TODO Task\n  SCHEDULED: <2024-01-15 Mon>\n- TODO Other")
			FullyEqual("Scheduled after a blank line", "TODO Task\n\nSCHEDULED: <2024-01-15 Mon>")

			// The day name is regenerated from the date, so a wrong one is
			// corrected on the way out.
			Varies("Scheduled with the wrong day name", "TODO Task\nSCHEDULED: <2024-01-15 Fri>", "TODO Task\nSCHEDULED: <2024-01-15 Mon>")

			// Syntax that is not modelled stays as text so nothing is lost.
			FullyEqual("Scheduled with a time range", "TODO Task\nSCHEDULED: <2024-01-15 Mon 10:00-11:00>")
			FullyEqual("Scheduled with a plain date", "TODO Task\nSCHEDULED: 2024-01-15")
		})
	})

	Describe("Nested page links", func() {
		// Given a nested page link, When it is written, Then no backslash is added
		FullyEqual("Page link with a page link inside", "- [[a [[b]]]]")
		FullyEqual("Extended hashtag with a page link inside", "- #[[a [[b]]]]")
		FullyEqual("Page link with two page links inside", "- [[a [[b]] c [[d]]]]")
		FullyEqual("Page link with page links nested two levels", "- [[a [[b [[c]]]]]]")
		FullyEqual("Page link with brackets that do not balance", "- [[a [[b]]")
		FullyEqual("Page link with a label and a page link inside", "- [x]([[a [[b]]]])")
		FullyEqual("Embed of a page link with a page link inside", "- {{embed [[a [[b]]]]}}")
	})

	Describe("Hashtags at the end of emphasis", func() {
		// Given emphasis that ends with a hashtag, When it is written, Then it is unchanged
		FullyEqual("Italic text that ends with a hashtag", "- *see #x*")
		FullyEqual("Bold text that ends with a hashtag", "- **Status: #done**")
		FullyEqual("Bold text with a hashtag in the middle", "- **a #x b**")
		FullyEqual("Strikethrough text that ends with a hashtag", "- ~~a #x~~")
		FullyEqual("Highlighted text that ends with a hashtag", "- ^^a #x^^")
		FullyEqual("Italic text that ends with a bracketed hashtag", "- *see #[[a b]]*")
	})

	Describe("Block properties after a quote", func() {
		// Given a quote followed by an id property, When it is written, Then
		// it is unchanged and the property is not written as part of the quote
		FullyEqual("Id property after a quote", "- > quoted\n  id:: 65e5a0b2-1c2d-4e3f-9a8b-0c1d2e3f4a5b")
		FullyEqual("Properties after a quote with a child", "- > quoted\n  id:: 65e5a0b2-1c2d-4e3f-9a8b-0c1d2e3f4a5b\n  collapsed:: true\n\t- child")
		FullyEqual("Id property after a list with stars", "- * star\n  id:: 65e5a0b2-1c2d-4e3f-9a8b-0c1d2e3f4a5b")
		FullyEqual("Id property after an ordered list", "- 1. one\n  id:: 65e5a0b2-1c2d-4e3f-9a8b-0c1d2e3f4a5b")
		FullyEqual("Id property after HTML", "- <div>x</div>\n  id:: 65e5a0b2-1c2d-4e3f-9a8b-0c1d2e3f4a5b")
		FullyEqual("Properties after a table", "- | a   | b   |\n  | --- | --- |\n  | 1   | 2   |\n  id:: 65e5a0b2-1c2d-4e3f-9a8b-0c1d2e3f4a5b\n  collapsed:: true\n\t- child")

		// A line that is not a property is still a part of the quote
		Varies("Lazy line after a quote", "- > a\n  b", "- > a\n  > b")
	})
})

var _ = Describe("Blank lines before a paragraph that ends in properties", func() {
	FullyEqual("keeps the blank line between two paragraphs when properties follow", "- a\n  b\n\n  c\n  d:: e")
	FullyEqual("keeps the blank line after the first line when properties follow", "- a\n\n  c\n  d:: e")
	FullyEqual("adds no blank line before a paragraph that ends in properties", "- a\n  c\n  d:: e")
	FullyEqual("keeps the blank line between paragraphs when an id follows", "- first\n\n  second\n  id:: 65e5a8b0-0000-4000-8000-000000000000")
	FullyEqual("keeps the blank line after a scheduled date when properties follow", "- TODO first\n  SCHEDULED: <2024-01-01 Mon>\n\n  second\n  key:: v")
	FullyEqual("keeps the blank line between page paragraphs when properties follow", "first\n\nsecond\nkey:: v")
	FullyEqual("keeps the blank line before properties that come before text", "- a\n\n  d:: e\n  c")
	FullyEqual("keeps the blank lines before a paragraph and before its properties", "- a\n\n  c\n\n  d:: e")
})

var _ = Describe("YAML front matter", func() {
	FullyEqual("keeps front matter with a title and a list of tags", "---\ntitle: Front\ntags: [a, b]\n---\n\n- b")
	FullyEqual("keeps front matter with a title and an alias", "---\ntitle: X\nalias: Y\n---\n\n- a")
	FullyEqual("keeps underscores and stars in a front matter value", "---\ntitle: My_Page *draft*\n---")
	FullyEqual("keeps a YAML list in front matter", "---\ntags:\n  - a\n  - b\n---\n\n- a")
	FullyEqual("keeps the indentation of a folded YAML value", "---\ndescription: >\n  long\n  text\n---\n\n- a")

	It("does not escape the brackets of a YAML list in front matter", func() {
		// Given front matter with a YAML flow list and no blank line after it
		out := parseAndOutput("---\ntags: [a, b]\n---\n- a\n")

		// Then the brackets are not escaped on save
		Expect(out).ToNot(ContainSubstring("\\["))
	})

	FullyEqual("keeps empty front matter", "---\n---\n- a")
	FullyEqual("keeps a first line of --- without an end line as a thematic break", "---\ntitle: X\n- a")

	It("writes the front matter from the properties when a property is changed", func() {
		// Given front matter with a title and an alias
		block, err := markdown.ParseString("---\ntitle: X\nalias: Y\n---\n\n- a")
		Expect(err).ToNot(HaveOccurred())

		// When the title is changed
		block.FindProperties().Set("title", content.NewText("Z"))

		// Then the front matter has the new title and keeps the alias
		out, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal("---\ntitle: Z\nalias: Y\n---\n\n- a"))
	})

	It("reads no properties from front matter with a YAML list", func() {
		// Given front matter where a line is not a name and a value
		block, err := markdown.ParseString("---\ntitle: X\ntags:\n  - a\n---\n\n- a")
		Expect(err).ToNot(HaveOccurred())

		// Then the page has no properties, the same as in Logseq
		Expect(block.FindProperties()).To(BeNil())
	})
})

var _ = Describe("Text of commands that is read as Markdown", func() {
	// Given a command whose text is not changed, When written, Then the text
	// is the same, also for lines that the writer writes in another way.
	FullyEqual("keeps a line that starts with a dash in a note", "- n\n  #+BEGIN_NOTE\n  see [[a]]\n  - b\n  #+END_NOTE")
	FullyEqual("keeps a line that starts with a hash in a note", "- n\n  #+BEGIN_NOTE\n  # not a heading\n  #+END_NOTE")
	FullyEqual("keeps a table in a quote as it is", "- n\n  #+BEGIN_QUOTE\n  | a | b |\n  | - | - |\n  | 1 | 2 |\n  #+END_QUOTE")
	FullyEqual("keeps a quote inside a note", "- n\n  #+BEGIN_NOTE\n  #+BEGIN_QUOTE\n  inner [[a]]\n  #+END_QUOTE\n  #+END_NOTE")

	It("changes only the line with the link when a page link in a note is changed", func() {
		// Given a note with a page link and a line that starts with a dash
		block, err := markdown.ParseString("- n\n  #+BEGIN_NOTE\n  see [[old]]\n  - b\n  #+END_NOTE")
		Expect(err).ToNot(HaveOccurred())

		// When the page link is pointed at another page
		refs := block.Children().PageReferences()
		Expect(refs).To(HaveLen(1))
		refs[0].(content.PageRef).SetTo("new")

		// Then only the line with the link changes
		out, err := markdown.AsString(block)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal("- n\n  #+BEGIN_NOTE\n  see [[new]]\n  - b\n  #+END_NOTE"))
	})

	It("reads no Markdown in an example", func() {
		// Given an example with a page link
		block, err := markdown.ParseString("- n\n  #+BEGIN_EXAMPLE\n  [[p]]\n  #+END_EXAMPLE")
		Expect(err).ToNot(HaveOccurred())

		// Then the example has no children and no references
		command := block.Children().FindDeep(content.IsOfType[*content.AdvancedCommand]()).(*content.AdvancedCommand)
		Expect(command.FirstChild()).To(BeNil())
		Expect(block.Children().PageReferences()).To(BeEmpty())
	})
})
