package markdown_test

import (
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func parseAndOutput(input string) string {
	block, err := markdown.ParseString(input)
	Expect(err).ToNot(HaveOccurred())
	v, err := markdown.AsString(block)
	Expect(err).ToNot(HaveOccurred())
	return v
}

func FullyEqual(name string, input string) {
	It(name, func() {
		v := parseAndOutput(input)
		Expect(v).To(Equal(input))
	})
}

func Varies(name string, input string, output string) {
	It(name, func() {
		v := parseAndOutput(input)
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
