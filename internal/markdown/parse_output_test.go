package markdown_test

import (
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

	Describe("Inline formatting", func() {
		FullyEqual("Bold text", "**Basic** content")
		FullyEqual("Bold text with newline", "**Basic\ncontent**")
		FullyEqual("Bold text with hard newline", "**Basic\\\ncontent**")
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
		FullyEqual("Hash followed by a space", "issue # 5")
		FullyEqual("Hash at the end of a word", "C# is fine")
		FullyEqual("Hash at the end of a line", "content #")
	})

	Describe("Heading", func() {
		FullyEqual("Heading 1", "# Heading")
		FullyEqual("Heading 2", "## Heading")
		FullyEqual("Heading 3", "### Heading")
		FullyEqual("Heading 4", "#### Heading")
		FullyEqual("Heading 5", "##### Heading")
		FullyEqual("Heading 6", "###### Heading")
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
		FullyEqual("Property with values that are escaped", "tags:: First, Second \\[escaped\\]")
		FullyEqualWhenReadWith("Property with values of a configured list property",
			"key:: First, Second Name",
			markdown.WithPropertiesSeparatedByCommas("key"),
		)
		FullyEqualWhenReadWith("Property whose references are ignored",
			"author:: [[Someone]], Another",
			markdown.WithIgnoredPageReferences("author"),
		)
		FullyEqual("Multiple properties", "key:: value\nkey2:: value2")
		FullyEqual("Properties followed by trailing paragraph", "key:: value\nParagraph")
		FullyEqual("Paragraphs interrupted by properties", "Paragraph\nkey:: value")
		FullyEqual("Paragraphs interrupted by properties followed by more paragraph", "Paragraph\nkey:: value\nParagraph")
		FullyEqual("Paragraph followed by properties", "Paragraph\n\nkey:: value")
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
			FullyEqual("TODO on its own", "TODO")
			FullyEqual("DONE on its own", "DONE")
			FullyEqual("TODO with sub block", "- TODO\n\t- child")
			Varies("TODO with trailing space", "TODO ", "TODO")

			// The marker takes up the whole first line, so what follows it moves
			// up to sit after the marker.
			Varies("TODO with text on the next line", "TODO\nmore", "TODO more")
			Varies("TODO followed by formatting", "TODO**bold**", "TODO **bold**")
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

		Describe("Footnotes", func() {
			FullyEqual("Footnote reference", "Task with a footnote[^1] in it")
			FullyEqual("Footnote definition", "[^1]: The footnote")
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
})
